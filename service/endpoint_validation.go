package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// hostRegexp matches a valid host. The reference RFCs consulted for this are the following ones:
// - https://datatracker.ietf.org/doc/html/rfc1034#section-3.5
// - https://datatracker.ietf.org/doc/html/rfc2181#section-11
// Even though the RFC1034 states that labels "must start with a letter, end with a letter or digit, and have as
// interior characters only letters, digits, and hyphen", there are domains which have labels that start or end with
// digits. For example: https://github.com/opendns/public-domain-lists/blob/master/opendns-random-domains.txt . Also,
// length validation on labels and FQDN is left to do it with "splits", to avoid complicating the regex even more.
var fqdnRegexp = regexp.MustCompile(`^(?:[a-zA-Z\d](?:[a-zA-Z\d-]*[a-zA-Z\d])?\.)+[a-zA-Z](?:[a-zA-Z\d-]*[a-zA-Z])?$`)

// schemeRegexp matches a valid scheme, as per RFC 3986
var schemeRegexp = regexp.MustCompile(`^[a-zA-Z][a-zA-Z\d+\-.]*$`)

// validAvailabilityStatuses is a map containing the valid availability statuses. It has this form because a map is
// faster for these lookups.
var validAvailabilityStatuses = map[string]struct{}{
	model.Available:   {},
	model.Unavailable: {},
}

const (
	defaultScheme    = "https"
	defaultPort      = 443
	defaultVerifySsl = false
	maxFqdnLength    = 255 // RFC2181
	maxLabelLength   = 63  // RFC2181
	maxPort          = 65535
)

func ValidateEndpointCreateRequest(dao dao.EndpointDao, ecr *model.EndpointCreateRequest) error {
	sourceId, err := util.InterfaceToInt64(ecr.SourceIDRaw)
	if err != nil {
		return fmt.Errorf("the provided source ID is not valid")
	}

	if sourceId < 1 {
		return fmt.Errorf("invalid source id")
	}

	ecr.SourceID = sourceId

	if ecr.Default && !dao.CanEndpointBeSetAsDefaultForSource(sourceId) {
		return fmt.Errorf("a default endpoint already exists for the provided source")
	}

	if !dao.SourceHasEndpoints(sourceId) {
		ecr.Default = true
	}

	if !dao.IsRoleUniqueForSource(ecr.Role, sourceId) {
		return fmt.Errorf("the role already exists for the given source")
	}

	if ecr.Scheme == nil || !schemeRegexp.MatchString(*ecr.Scheme) {
		tmp := defaultScheme
		ecr.Scheme = &tmp
	}

	if ecr.Host != "" {
		if utf8.RuneCountInString(ecr.Host) > maxFqdnLength {
			return fmt.Errorf("the provided host is longer than %d characters", maxFqdnLength)
		}

		if !fqdnRegexp.MatchString(ecr.Host) {
			return fmt.Errorf("the provided host is invalid")
		}

		labels := strings.Split(ecr.Host, ".")
		for _, label := range labels {
			if utf8.RuneCountInString(ecr.Host) > maxLabelLength {
				return fmt.Errorf("the label '%s' is greater than %d characters", label, maxLabelLength)
			}
		}
	}

	if ecr.Port == nil || *ecr.Port <= 0 {
		tmp := defaultPort
		ecr.Port = &tmp
	}

	if *ecr.Port > maxPort {
		return fmt.Errorf("invalid port number")
	}

	if ecr.VerifySsl == nil {
		tmp := defaultVerifySsl
		ecr.VerifySsl = &tmp
	}

	if *ecr.VerifySsl && (ecr.CertificateAuthority == nil || *ecr.CertificateAuthority == "") {
		return fmt.Errorf("the certificate authority cannot be empty")
	}

	// The team decided that the availability statuses will default to "in_progress" whenever they come empty, since
	// setting them as "unavailable" by default may lead to some confusion to the calling clients.
	if ecr.AvailabilityStatus == "" {
		ecr.AvailabilityStatus = model.InProgress
	} else {
		if _, ok := validAvailabilityStatuses[ecr.AvailabilityStatus]; !ok {
			return fmt.Errorf("invalid availability status")
		}
	}

	return nil
}

func ValidateEndpointEditRequest(dao dao.EndpointDao, sourceId int64, editRequest *model.EndpointEditRequest) error {
	if editRequest.Default != nil && (*editRequest.Default && !dao.CanEndpointBeSetAsDefaultForSource(sourceId)) {
		return fmt.Errorf("a default endpoint already exists for the provided source")
	}

	if editRequest.Role != nil && !dao.IsRoleUniqueForSource(*editRequest.Role, sourceId) {
		return fmt.Errorf("the role already exists for the given source")
	}

	if editRequest.Scheme == nil || (editRequest.Scheme != nil && !schemeRegexp.MatchString(*editRequest.Scheme)) {
		tmp := defaultScheme
		editRequest.Scheme = &tmp
	}

	if editRequest.Host != nil && *editRequest.Host != "" {
		if utf8.RuneCountInString(*editRequest.Host) > maxFqdnLength {
			return fmt.Errorf("the provided host is longer than %d characters", maxFqdnLength)
		}

		if !fqdnRegexp.MatchString(*editRequest.Host) {
			return fmt.Errorf("the provided host is invalid")
		}

		labels := strings.Split(*editRequest.Host, ".")
		for _, label := range labels {
			if utf8.RuneCountInString(*editRequest.Host) > maxLabelLength {
				return fmt.Errorf("the label '%s' is greater than %d characters", label, maxLabelLength)
			}
		}
	}

	if editRequest.Port == nil || (editRequest.Port != nil && *editRequest.Port <= 0) {
		tmp := defaultPort
		editRequest.Port = &tmp
	}

	if *editRequest.Port > maxPort {
		return fmt.Errorf("invalid port number")
	}

	if editRequest.VerifySsl == nil {
		tmp := defaultVerifySsl
		editRequest.VerifySsl = &tmp
	}

	if *editRequest.VerifySsl && (editRequest.CertificateAuthority == nil || *editRequest.CertificateAuthority == "") {
		return fmt.Errorf("the certificate authority cannot be empty")
	}

	if editRequest.AvailabilityStatus != nil {
		if _, ok := validAvailabilityStatuses[*editRequest.AvailabilityStatus]; !ok {
			return fmt.Errorf("invalid availability status")
		}
	}

	return nil
}
