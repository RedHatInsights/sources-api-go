#!/usr/bin/env python3

"""
Simple Sources API Test Script
Creates a source, verifies it, then deletes it. One-shot validation.

Use this for quick smoke tests before/after DR exercises.
"""

import argparse
import json
import os
import sys
import time
from typing import Optional

import requests


class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    NC = '\033[0m'


def log(level: str, message: str, color: str = Colors.NC):
    from datetime import datetime

    timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    print(f"{color}[{timestamp}] [{level}]{Colors.NC} {message}", flush=True)


def get_auth_headers(auth_token: str, auth_type: str) -> dict:
    if auth_type == "jwt":
        return {
            'Authorization': f'Bearer {auth_token}',
            'Content-Type': 'application/json',
        }
    return {
        'x-rh-identity': auth_token,
        'Content-Type': 'application/json',
    }


def get_request_id(response: requests.Response) -> Optional[str]:
    for header in [
        'x-rh-insights-request-id',
        'x-rh-request-id',
        'x-request-id',
    ]:
        val = response.headers.get(header)
        if val:
            return val
    return None


def run_test(
    api_url: str,
    auth_token: str,
    auth_type: str = "identity",
    source_type_id: Optional[str] = None,
    proxy: Optional[str] = None,
    keep: bool = False,
) -> bool:
    """Run a single create-verify-delete cycle. Returns True on success."""
    api_url = api_url.rstrip('/')
    headers = get_auth_headers(auth_token, auth_type)
    proxies = {'http': proxy, 'https': proxy} if proxy else None
    success = True

    log("INFO", f"API URL: {api_url}", Colors.BLUE)
    log("INFO", f"Auth type: {auth_type}", Colors.BLUE)
    if proxy:
        log("INFO", f"Proxy: {proxy}", Colors.BLUE)
    print()

    # Step 1: List existing sources
    log("INFO", "Step 1: Listing existing sources...", Colors.BLUE)
    try:
        resp = requests.get(
            f"{api_url}/sources",
            headers=headers,
            params={"limit": 5},
            timeout=15,
            proxies=proxies,
        )
        if resp.status_code == 200:
            data = resp.json()
            count = data.get('meta', {}).get('count', '?')
            log("SUCCESS", f"Listed sources: {count} total", Colors.GREEN)
        else:
            rid = get_request_id(resp)
            log(
                "ERROR",
                f"List failed: HTTP {resp.status_code} {resp.text[:200]}",
                Colors.RED,
            )
            if rid:
                log("ERROR", f"  request_id: {rid}", Colors.RED)
            success = False
    except requests.RequestException as e:
        log("ERROR", f"List failed: {e}", Colors.RED)
        success = False

    # Step 2: Create a test source
    log("INFO", "Step 2: Creating test source...", Colors.BLUE)
    source_id = None
    source_name = f"dr-simple-test-{int(time.time())}"
    payload = {"name": source_name}
    if source_type_id:
        payload["source_type_id"] = source_type_id

    try:
        resp = requests.post(
            f"{api_url}/sources",
            headers=headers,
            json=payload,
            timeout=30,
            proxies=proxies,
        )
        if resp.status_code == 201:
            data = resp.json()
            source_id = data.get('id')
            log(
                "SUCCESS",
                f"Created source: id={source_id} name={source_name}",
                Colors.GREEN,
            )
            log(
                "INFO",
                f"  Response: {json.dumps(data, indent=2)[:500]}",
                Colors.BLUE,
            )
        else:
            rid = get_request_id(resp)
            log(
                "ERROR",
                f"Create failed: HTTP {resp.status_code} {resp.text[:200]}",
                Colors.RED,
            )
            if rid:
                log("ERROR", f"  request_id: {rid}", Colors.RED)
            return False
    except requests.RequestException as e:
        log("ERROR", f"Create failed: {e}", Colors.RED)
        return False

    # Step 3: Verify source by ID
    log("INFO", f"Step 3: Verifying source {source_id}...", Colors.BLUE)
    try:
        resp = requests.get(
            f"{api_url}/sources/{source_id}",
            headers=headers,
            timeout=10,
            proxies=proxies,
        )
        if resp.status_code == 200:
            data = resp.json()
            if data.get('name') == source_name:
                log(
                    "SUCCESS",
                    f"Verified: name matches ({source_name})",
                    Colors.GREEN,
                )
            else:
                log(
                    "ERROR",
                    f"Name mismatch: expected '{source_name}', "
                    f"got '{data.get('name')}'",
                    Colors.RED,
                )
                success = False
        else:
            rid = get_request_id(resp)
            log(
                "ERROR",
                f"Verify failed: HTTP {resp.status_code}",
                Colors.RED,
            )
            if rid:
                log("ERROR", f"  request_id: {rid}", Colors.RED)
            success = False
    except requests.RequestException as e:
        log("ERROR", f"Verify failed: {e}", Colors.RED)
        success = False

    # Step 4: Verify source appears in list
    log(
        "INFO",
        f"Step 4: Verifying source {source_id} appears in list...",
        Colors.BLUE,
    )
    try:
        resp = requests.get(
            f"{api_url}/sources",
            headers=headers,
            params={"limit": 100},
            timeout=15,
            proxies=proxies,
        )
        if resp.status_code == 200:
            sources = resp.json().get('data', [])
            found = any(str(s.get('id')) == str(source_id) for s in sources)
            if found:
                log(
                    "SUCCESS",
                    f"Source {source_id} found in list",
                    Colors.GREEN,
                )
            else:
                log(
                    "WARNING",
                    f"Source {source_id} not in first 100 results "
                    "(may be pagination)",
                    Colors.YELLOW,
                )
    except requests.RequestException as e:
        log("WARNING", f"List verification failed: {e}", Colors.YELLOW)

    # Step 5: Delete (unless --keep)
    if keep:
        log(
            "INFO",
            f"Step 5: Skipping delete (--keep). Source {source_id} preserved.",
            Colors.BLUE,
        )
    else:
        log("INFO", f"Step 5: Deleting source {source_id}...", Colors.BLUE)
        try:
            resp = requests.delete(
                f"{api_url}/sources/{source_id}",
                headers=headers,
                timeout=15,
                proxies=proxies,
            )
            if resp.status_code in (200, 204):
                log(
                    "SUCCESS",
                    f"Deleted source {source_id}",
                    Colors.GREEN,
                )
            elif resp.status_code == 404:
                log(
                    "WARNING",
                    f"Source {source_id} already gone (404)",
                    Colors.YELLOW,
                )
            else:
                rid = get_request_id(resp)
                log(
                    "ERROR",
                    f"Delete failed: HTTP {resp.status_code}",
                    Colors.RED,
                )
                if rid:
                    log("ERROR", f"  request_id: {rid}", Colors.RED)
                success = False
        except requests.RequestException as e:
            log("ERROR", f"Delete failed: {e}", Colors.RED)
            success = False

    # Summary
    print()
    if success:
        log("SUCCESS", "All checks passed!", Colors.GREEN)
    else:
        log("ERROR", "Some checks failed — see above.", Colors.RED)

    return success


def main():
    parser = argparse.ArgumentParser(
        description=(
            "Simple Sources API Test — create, verify, delete. "
            "Quick smoke test for DR exercises."
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Local
  %(prog)s --url http://localhost:3000/api/sources/v3.1

  # Stage with JWT
  %(prog)s \\
      --url https://console.stage.redhat.com/api/sources/v3.1 \\
      --auth-type jwt \\
      --auth-token "$JWT_TOKEN" \\
      --proxy http://squid.corp.redhat.com:3128

  # Keep the source (no cleanup)
  %(prog)s --keep
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
            'eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiIxMjM0NSIsIm9y'
            'Z19pZCI6IjEyMzQ1IiwidHlwZSI6IlVzZXIiLCJ1c2VyIjp7ImlzX29y'
            'Z19hZG1pbiI6dHJ1ZX0sImludGVybmFsIjp7Im9yZ19pZCI6IjEyMzQ1'
            'In19fQo=',
        ),
        help=(
            'Auth token '
            '(default: $AUTH_TOKEN, $RH_IDENTITY, or test identity)'
        ),
    )

    parser.add_argument(
        '--auth-type',
        choices=['identity', 'jwt'],
        default=os.environ.get('AUTH_TYPE', 'identity'),
        help='Auth type (default: $AUTH_TYPE or identity)',
    )

    parser.add_argument(
        '--source-type-id',
        default=os.environ.get('SOURCE_TYPE_ID'),
        help='Source type ID (default: $SOURCE_TYPE_ID or none)',
    )

    parser.add_argument(
        '--keep',
        action='store_true',
        help='Keep the test source (do not delete)',
    )

    parser.add_argument(
        '--proxy',
        default=os.environ.get('HTTP_PROXY')
        or os.environ.get('HTTPS_PROXY'),
        help='HTTP/HTTPS proxy URL (default: $HTTP_PROXY or $HTTPS_PROXY)',
    )

    args = parser.parse_args()

    ok = run_test(
        api_url=args.url,
        auth_token=args.auth_token,
        auth_type=args.auth_type,
        source_type_id=args.source_type_id,
        proxy=args.proxy,
        keep=args.keep,
    )

    sys.exit(0 if ok else 1)


if __name__ == '__main__':
    main()
