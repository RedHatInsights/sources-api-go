#!/usr/bin/env python3

"""
Continuous Sources API Synthetic Traffic Script
Creates sources every 20s, lists/checks sources every 5s, cleans up on exit.

Designed for Database Recovery (DR) exercises to generate realistic traffic
against the Sources API while validating data integrity before/after failover.
"""

import argparse
import json
import os
import signal
import sys
import threading
import time
import uuid
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional

import requests


class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    CYAN = '\033[0;36m'
    NC = '\033[0m'


@dataclass
class SourceRecord:
    """Track a created source"""
    source_id: str
    name: str
    created_at: float
    verified: bool = False
    deleted: bool = False


class SourcesTester:
    """Continuous Sources API tester for DR exercises"""

    # Prefix for all test sources — used for identification and cleanup
    SOURCE_NAME_PREFIX = "dr-synthetic-test"

    def __init__(
        self,
        api_url: str,
        auth_token: str,
        auth_type: str = "identity",
        create_interval: int = 20,
        check_interval: int = 5,
        cleanup_on_exit: bool = True,
        log_dir: str = "./test_logs",
        proxy: Optional[str] = None,
        source_type_id: Optional[str] = None,
    ):
        self.api_url = api_url.rstrip('/')
        self.auth_token = auth_token
        self.auth_type = auth_type.lower()
        self.create_interval = create_interval
        self.check_interval = check_interval
        self.cleanup_on_exit = cleanup_on_exit
        self.log_dir = Path(log_dir)
        self.source_type_id = source_type_id

        self.proxies = None
        if proxy:
            self.proxies = {'http': proxy, 'https': proxy}

        if self.auth_type not in ("identity", "jwt"):
            raise ValueError(
                f"Invalid auth_type '{auth_type}'. Must be 'identity' or 'jwt'"
            )

        self.log_dir.mkdir(exist_ok=True)

        # Tracking
        self.tracking_file = self.log_dir / "sources.json"
        self.sources: Dict[str, SourceRecord] = {}
        self.sources_lock = threading.Lock()
        self.running = True

        # Statistics
        self.stats = {
            'created': 0,
            'create_failed': 0,
            'listed': 0,
            'list_failed': 0,
            'verified': 0,
            'verify_failed': 0,
            'deleted': 0,
            'delete_failed': 0,
            'integrity_ok': 0,
            'integrity_fail': 0,
        }

        self._load_tracking()

        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _get_auth_headers(self) -> dict:
        if self.auth_type == "jwt":
            return {
                'Authorization': f'Bearer {self.auth_token}',
                'Content-Type': 'application/json',
            }
        return {
            'x-rh-identity': self.auth_token,
            'Content-Type': 'application/json',
        }

    def _log(self, level: str, message: str, color: str = Colors.NC):
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        print(
            f"{color}[{timestamp}] [{level}]{Colors.NC} {message}", flush=True
        )

    def log_info(self, message: str):
        self._log("INFO", message, Colors.BLUE)

    def log_success(self, message: str):
        self._log("SUCCESS", message, Colors.GREEN)

    def log_error(self, message: str, request_id: Optional[str] = None):
        if request_id:
            message = f"{message} [request_id: {request_id}]"
        self._log("ERROR", message, Colors.RED)

    def log_warning(self, message: str):
        self._log("WARNING", message, Colors.YELLOW)

    def log_detail(self, message: str):
        self._log("DETAIL", message, Colors.CYAN)

    def _get_request_id(self, response: requests.Response) -> Optional[str]:
        for header in [
            'x-rh-insights-request-id',
            'x-rh-request-id',
            'x-request-id',
        ]:
            request_id = response.headers.get(header)
            if request_id:
                return request_id
        return None

    def _signal_handler(self, signum, frame):
        self.log_warning("Shutting down...")
        self.running = False

    def _load_tracking(self):
        if self.tracking_file.exists():
            try:
                with open(self.tracking_file, 'r') as f:
                    data = json.load(f)
                    for source_id, record in data.items():
                        self.sources[source_id] = SourceRecord(**record)
                self.log_info(f"Loaded {len(self.sources)} tracked sources")
            except Exception as e:
                self.log_error(f"Failed to load tracking file: {e}")

    def _save_tracking(self):
        try:
            with self.sources_lock:
                data = {
                    source_id: {
                        'source_id': rec.source_id,
                        'name': rec.name,
                        'created_at': rec.created_at,
                        'verified': rec.verified,
                        'deleted': rec.deleted,
                    }
                    for source_id, rec in self.sources.items()
                }
            with open(self.tracking_file, 'w') as f:
                json.dump(data, f, indent=2)
        except Exception as e:
            self.log_error(f"Failed to save tracking file: {e}")

    def _generate_source_name(self) -> str:
        short_id = uuid.uuid4().hex[:8]
        timestamp = datetime.utcnow().strftime('%Y%m%d-%H%M%S')
        return f"{self.SOURCE_NAME_PREFIX}-{timestamp}-{short_id}"

    def create_source(self):
        """Create a new source via POST /sources"""
        try:
            name = self._generate_source_name()
            self.log_info(f"Creating source: {name}")

            payload = {"name": name}
            if self.source_type_id:
                payload["source_type_id"] = self.source_type_id

            headers = self._get_auth_headers()
            response = requests.post(
                f"{self.api_url}/sources",
                headers=headers,
                json=payload,
                timeout=30,
                proxies=self.proxies,
            )

            if response.status_code != 201:
                request_id = self._get_request_id(response)
                self.log_error(
                    f"Failed to create source: HTTP {response.status_code} "
                    f"— {response.text[:200]}",
                    request_id,
                )
                self.stats['create_failed'] += 1
                return

            data = response.json()
            source_id = data.get('id')

            if not source_id:
                request_id = self._get_request_id(response)
                self.log_error(f"No source ID in response: {data}", request_id)
                self.stats['create_failed'] += 1
                return

            record = SourceRecord(
                source_id=str(source_id),
                name=name,
                created_at=time.time(),
            )

            with self.sources_lock:
                self.sources[str(source_id)] = record

            self.stats['created'] += 1
            self.log_success(f"Created source: id={source_id} name={name}")

            # Save response for audit
            log_file = self.log_dir / f"source-{source_id}.json"
            with open(log_file, 'w') as f:
                json.dump(data, f, indent=2)

            self._save_tracking()

        except requests.RequestException as e:
            request_id = None
            if hasattr(e, 'response') and e.response is not None:
                request_id = self._get_request_id(e.response)
            self.log_error(f"Network error creating source: {e}", request_id)
            self.stats['create_failed'] += 1
        except Exception as e:
            self.log_error(f"Error creating source: {e}")
            self.stats['create_failed'] += 1

    def check_sources(self):
        """List sources and verify created ones exist (GET /sources)"""
        try:
            headers = self._get_auth_headers()
            response = requests.get(
                f"{self.api_url}/sources",
                headers=headers,
                params={"limit": 100},
                timeout=15,
                proxies=self.proxies,
            )

            if response.status_code != 200:
                request_id = self._get_request_id(response)
                self.log_error(
                    f"Failed to list sources: HTTP {response.status_code}",
                    request_id,
                )
                self.stats['list_failed'] += 1
                return

            data = response.json()
            api_sources = data.get('data', [])
            meta = data.get('meta', {})
            total_count = meta.get('count', len(api_sources))

            self.stats['listed'] += 1
            self.log_info(
                f"Listed sources: {len(api_sources)} returned, "
                f"{total_count} total"
            )

            # Build set of source IDs from API response
            api_source_ids = {str(s.get('id')) for s in api_sources}

            # Verify our tracked (non-deleted) sources exist in the listing
            with self.sources_lock:
                unverified = [
                    rec
                    for rec in self.sources.values()
                    if not rec.deleted and not rec.verified
                ]

            for rec in unverified:
                if rec.source_id in api_source_ids:
                    rec.verified = True
                    self.stats['verified'] += 1
                    self.log_success(
                        f"Verified source {rec.source_id} ({rec.name})"
                    )

        except requests.RequestException as e:
            self.stats['list_failed'] += 1
        except Exception as e:
            self.log_error(f"Error checking sources: {e}")
            self.stats['list_failed'] += 1

    def verify_source_by_id(self, source_id: str) -> bool:
        """Verify a single source exists via GET /sources/{id}"""
        try:
            headers = self._get_auth_headers()
            response = requests.get(
                f"{self.api_url}/sources/{source_id}",
                headers=headers,
                timeout=10,
                proxies=self.proxies,
            )

            if response.status_code == 200:
                data = response.json()
                returned_name = data.get('name', '')
                self.stats['integrity_ok'] += 1
                self.log_detail(
                    f"Source {source_id} integrity OK: name={returned_name}"
                )
                return True

            if response.status_code == 404:
                self.stats['integrity_fail'] += 1
                self.log_error(
                    f"Source {source_id} NOT FOUND — data integrity issue!"
                )
                return False

            request_id = self._get_request_id(response)
            self.log_error(
                f"Unexpected response for source {source_id}: "
                f"HTTP {response.status_code}",
                request_id,
            )
            return False

        except requests.RequestException:
            return False
        except Exception as e:
            self.log_error(f"Error verifying source {source_id}: {e}")
            return False

    def integrity_check(self):
        """Verify all tracked sources still exist — key for DR validation"""
        with self.sources_lock:
            active = [
                rec for rec in self.sources.values() if not rec.deleted
            ]

        if not active:
            return

        self.log_info(
            f"Running integrity check on {len(active)} tracked sources..."
        )

        ok_count = 0
        fail_count = 0
        for rec in active:
            if not self.running:
                break
            if self.verify_source_by_id(rec.source_id):
                ok_count += 1
            else:
                fail_count += 1

        if fail_count > 0:
            self.log_error(
                f"Integrity check: {ok_count} OK, {fail_count} MISSING"
            )
        else:
            self.log_success(
                f"Integrity check: all {ok_count} sources present"
            )

    def delete_source(self, source_id: str) -> bool:
        """Delete a source via DELETE /sources/{id}"""
        try:
            headers = self._get_auth_headers()
            response = requests.delete(
                f"{self.api_url}/sources/{source_id}",
                headers=headers,
                timeout=15,
                proxies=self.proxies,
            )

            if response.status_code in (200, 204):
                self.stats['deleted'] += 1
                self.log_info(f"Deleted source {source_id}")
                return True

            if response.status_code == 404:
                self.log_warning(
                    f"Source {source_id} already deleted (404)"
                )
                return True

            request_id = self._get_request_id(response)
            self.log_error(
                f"Failed to delete source {source_id}: "
                f"HTTP {response.status_code}",
                request_id,
            )
            self.stats['delete_failed'] += 1
            return False

        except requests.RequestException as e:
            self.log_error(f"Network error deleting source {source_id}: {e}")
            self.stats['delete_failed'] += 1
            return False

    def cleanup(self):
        """Delete all test sources created by this script"""
        with self.sources_lock:
            to_delete = [
                rec for rec in self.sources.values() if not rec.deleted
            ]

        if not to_delete:
            self.log_info("No sources to clean up")
            return

        self.log_info(f"Cleaning up {len(to_delete)} test sources...")

        for rec in to_delete:
            if self.delete_source(rec.source_id):
                rec.deleted = True

        self._save_tracking()
        self.log_success("Cleanup complete")

    # --- Background threads ---

    def _create_loop(self):
        while self.running:
            self.create_source()
            for _ in range(self.create_interval):
                if not self.running:
                    break
                time.sleep(1)

    def _check_loop(self):
        while self.running:
            time.sleep(self.check_interval)
            if self.running:
                self.check_sources()

    def _integrity_loop(self):
        """Run integrity checks every 60 seconds"""
        while self.running:
            time.sleep(60)
            if self.running:
                self.integrity_check()

    def print_summary(self):
        self.log_info("=" * 50)
        self.log_info("Test run summary:")
        self.log_info(f"  Sources created:     {self.stats['created']}")
        self.log_info(f"  Create failures:     {self.stats['create_failed']}")
        self.log_info(f"  List calls:          {self.stats['listed']}")
        self.log_info(f"  List failures:       {self.stats['list_failed']}")
        self.log_info(f"  Sources verified:    {self.stats['verified']}")
        self.log_info(f"  Integrity OK:        {self.stats['integrity_ok']}")
        self.log_info(f"  Integrity FAIL:      {self.stats['integrity_fail']}")
        self.log_info(f"  Sources deleted:     {self.stats['deleted']}")
        self.log_info(f"  Delete failures:     {self.stats['delete_failed']}")

        with self.sources_lock:
            active = sum(1 for r in self.sources.values() if not r.deleted)
        self.log_info(f"  Active sources:      {active}")
        self.log_info(f"  Log dir:             {self.log_dir}")
        self.log_info("=" * 50)

    def run(self):
        self.log_info("Starting continuous Sources API synthetic traffic...")
        self.log_info("Configuration:")
        self.log_info(f"  API URL:           {self.api_url}")
        self.log_info(f"  Auth type:         {self.auth_type}")
        if self.proxies:
            self.log_info(f"  Proxy:             {self.proxies.get('http')}")
        if self.source_type_id:
            self.log_info(f"  Source type ID:     {self.source_type_id}")
        self.log_info(f"  Create interval:   {self.create_interval}s")
        self.log_info(f"  Check interval:    {self.check_interval}s")
        self.log_info(f"  Cleanup on exit:   {self.cleanup_on_exit}")
        self.log_info(f"  Logs:              {self.log_dir}")
        self.log_info("")
        self.log_info(
            "Traffic pattern: create source every "
            f"{self.create_interval}s, list/verify every "
            f"{self.check_interval}s, integrity check every 60s"
        )
        self.log_info("")
        self.log_info("Press Ctrl+C to stop")
        self.log_info("-" * 50)

        threads = [
            threading.Thread(
                target=self._create_loop, name="Creator", daemon=True
            ),
            threading.Thread(
                target=self._check_loop, name="Checker", daemon=True
            ),
            threading.Thread(
                target=self._integrity_loop, name="Integrity", daemon=True
            ),
        ]

        for thread in threads:
            thread.start()

        try:
            while self.running:
                time.sleep(1)
        except KeyboardInterrupt:
            self.running = False

        self.log_info("Waiting for threads to finish...")
        for thread in threads:
            thread.join(timeout=2)

        if self.cleanup_on_exit:
            self.cleanup()

        self._save_tracking()
        self.print_summary()


def main():
    parser = argparse.ArgumentParser(
        description="Continuous Sources API Synthetic Traffic for DR Exercises",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Local development
  %(prog)s --url http://localhost:3000/api/sources/v3.1

  # Stage with JWT auth
  %(prog)s \\
      --url https://console.stage.redhat.com/api/sources/v3.1 \\
      --auth-type jwt \\
      --auth-token "$JWT_TOKEN" \\
      --proxy http://squid.corp.redhat.com:3128

  # Stage with x-rh-identity (default)
  %(prog)s \\
      --url https://console.stage.redhat.com/api/sources/v3.1 \\
      --auth-token "$RH_IDENTITY"

  # No cleanup on exit (keep sources for post-DR verification)
  %(prog)s --no-cleanup
        """,
    )

    parser.add_argument(
        '--url',
        default=os.environ.get(
            'SOURCES_URL', 'http://localhost:3000/api/sources/v3.1'
        ),
        help=(
            'Sources API URL '
            '(default: $SOURCES_URL or http://localhost:3000/api/sources/v3.1)'
        ),
    )

    parser.add_argument(
        '--auth-token',
        default=os.environ.get('AUTH_TOKEN')
        or os.environ.get(
            'RH_IDENTITY',
            # Default test identity for local dev
            'eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiIxMjM0NSIsIm9y'
            'Z19pZCI6IjEyMzQ1IiwidHlwZSI6IlVzZXIiLCJ1c2VyIjp7ImlzX29y'
            'Z19hZG1pbiI6dHJ1ZX0sImludGVybmFsIjp7Im9yZ19pZCI6IjEyMzQ1'
            'In19fQo=',
        ),
        help=(
            'Auth token — x-rh-identity value or JWT token '
            '(default: $AUTH_TOKEN, $RH_IDENTITY, or test identity)'
        ),
    )

    parser.add_argument(
        '--auth-type',
        choices=['identity', 'jwt'],
        default=os.environ.get('AUTH_TYPE', 'identity'),
        help=(
            'Auth type: "identity" for x-rh-identity header, '
            '"jwt" for Authorization: Bearer '
            '(default: $AUTH_TYPE or identity)'
        ),
    )

    parser.add_argument(
        '--create-interval',
        type=int,
        default=20,
        help='Seconds between creating sources (default: 20)',
    )

    parser.add_argument(
        '--check-interval',
        type=int,
        default=5,
        help='Seconds between listing/checking sources (default: 5)',
    )

    parser.add_argument(
        '--source-type-id',
        default=os.environ.get('SOURCE_TYPE_ID'),
        help=(
            'Source type ID to use when creating sources '
            '(default: $SOURCE_TYPE_ID or none)'
        ),
    )

    parser.add_argument(
        '--no-cleanup',
        action='store_true',
        help='Do not delete test sources on exit (keep for DR verification)',
    )

    parser.add_argument(
        '--log-dir',
        default='./test_logs',
        help='Directory for log files (default: ./test_logs)',
    )

    parser.add_argument(
        '--proxy',
        default=os.environ.get('HTTP_PROXY')
        or os.environ.get('HTTPS_PROXY'),
        help=(
            'HTTP/HTTPS proxy URL (e.g., http://squid.corp.redhat.com:3128) '
            '(default: $HTTP_PROXY or $HTTPS_PROXY)'
        ),
    )

    args = parser.parse_args()

    tester = SourcesTester(
        api_url=args.url,
        auth_token=args.auth_token,
        auth_type=args.auth_type,
        create_interval=args.create_interval,
        check_interval=args.check_interval,
        cleanup_on_exit=not args.no_cleanup,
        log_dir=args.log_dir,
        proxy=args.proxy,
        source_type_id=args.source_type_id,
    )

    tester.run()


if __name__ == '__main__':
    main()
