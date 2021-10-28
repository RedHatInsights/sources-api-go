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
var validAvailabilityStatuses = []string{"", model.Available, model.Unavailable}

const (
	defaultScheme    = "https"
	defaultPort      = int64(443)
	defaultVerifySsl = true
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

	if ecr.Default == nil {
		return fmt.Errorf("default body parameter not provided")
	}

	if *ecr.Default && !dao.CanEndpointBeSetAsDefaultForSource(sourceId) {
		return fmt.Errorf("a default endpoint already exists for the provided source")
	}

	if !dao.SourceHasEndpoints(sourceId) {
		*ecr.Default = true
	}

	if ecr.ReceptorNode == nil || *ecr.ReceptorNode == "" {
		return fmt.Errorf("the receptor node cannot be empty")
	}

	if ecr.Role == nil || *ecr.Role == "" {
		return fmt.Errorf("role cannot be empty")
	}

	if !dao.IsRoleUniqueForSource(*ecr.Role, sourceId) {
		return fmt.Errorf("the role already exists for the given source")
	}

	if ecr.Scheme == nil || !schemeRegexp.MatchString(*ecr.Scheme) {
		tmp := defaultScheme
		ecr.Scheme = &tmp
	}

	if ecr.Host == nil || *ecr.Host == "" {
		return fmt.Errorf("the host cannot be empty")
	}

	if utf8.RuneCountInString(*ecr.Host) > maxFqdnLength {
		return fmt.Errorf("the provided host is longer than %d characters", maxFqdnLength)
	}

	if !fqdnRegexp.MatchString(*ecr.Host) {
		return fmt.Errorf("the provided host is invalid")
	}

	labels := strings.Split(*ecr.Host, ".")
	for _, label := range labels {
		if utf8.RuneCountInString(*ecr.Host) > maxLabelLength {
			return fmt.Errorf("the label '%s' is greater than %d characters", label, maxLabelLength)
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

	if !util.SliceContainsString(validAvailabilityStatuses, ecr.AvailabilityStatus) {
		return fmt.Errorf("invalid availability status")
	}

	return nil
}
