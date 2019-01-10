package protocols

import (
	"fmt"
	"net/url"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"unsafe"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	// DefaultAwsRegion represents the default AWS region for accessing AWS objects
	DefaultAwsRegion     = "us-west-2"
	awsRegionRegexpStr = "https?://s3-([^.]+)\\.amazonaws\\.com"
)
var awsRegionRegexp = regexp.MustCompile(awsRegionRegexpStr)

// RegionFromEndpoint attempts to extract an AWS region from the specified endpoint
// URL, returning an error if none can be found.
func RegionFromEndpoint(endpoint *url.URL) (*string, error) {
	if endpoint == nil {
		return nil, fmt.Errorf("can't extract region from nil endpoint")
	}
	matches := awsRegionRegexp.FindStringSubmatch(endpoint.String())
	if len(matches) == 2 {
		regionStr := matches[1]
		return &regionStr, nil
	}
	return nil, fmt.Errorf("no AWS region found in endpoint URL %v", endpoint)
}

// IsEC2 returns true if the current system appears to be an EC2 host, false
// otherwise.
func IsEC2() (bool, error) {
	// TODO: something less dumb
	// - https://stackoverflow.com/questions/54119890/how-do-i-determine-whether-my-application-is-running-in-amazon-ec2-without-net
	out, err := exec.Command("uname", "-a").Output()
	if err != nil {
		return false, err
	}
	uname := string(out)
	if strings.Contains(uname, "amzn") {
		return true, nil
	}
	return false, nil
}

// InitS3Session returns a new AWS session configured for S3 access via the specified endpoint and region.
// The credentialsChainVerboseErrors controls whether to return verbose error messages in the event AWS
// credentials cannot be determined.
func InitS3Session(endpointP *string, regionStrP *string, credentialsChainVerboseErrors bool) (*session.Session, error) {
	s3Config := aws.Config{
		Endpoint:                      endpointP,
		Region:                        regionStrP,
		S3ForcePathStyle:              aws.Bool(true),
		CredentialsChainVerboseErrors: aws.Bool(credentialsChainVerboseErrors),
	}
	s3Opts := session.Options{
		Config:            s3Config,
		SharedConfigState: session.SharedConfigEnable,
	}
	awsSession, err := session.NewSessionWithOptions(s3Opts)
	if err != nil {
		return nil, err
	}
	return awsSession, nil
}

// ValidateCredentials uses reflection to check whether we're falling back to IAM credentials
// TODO: https://github.com/aws/aws-sdk-go/issues/2392
func ValidateCredentials(awsSession *session.Session) (*session.Session, error) {
	providerVal := reflect.ValueOf(*awsSession.Config.Credentials).FieldByName("provider").Elem()
	if providerVal.Type() == reflect.TypeOf((*credentials.ChainProvider)(nil)) {
		chainProvider := (*credentials.ChainProvider)(unsafe.Pointer(providerVal.Pointer()))
		providers := chainProvider.Providers
		if len(providers) > 0 {
			err := reflect.ValueOf(providers[0]).Elem().FieldByName("Err")
			if err.IsValid() {
				if e2, ok := err.Interface().(error); ok {
					return nil, e2
				}
			}
		}
	}
	return awsSession, nil
}
