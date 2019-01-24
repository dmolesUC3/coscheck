package objects

import (
	"fmt"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/dmolesUC3/cos/internal/logging"
	"github.com/dmolesUC3/cos/internal/protocols"
)

// S3Object is an S3 implementation of Object
type S3Object struct {
	region     string
	endpoint   *url.URL
	bucket     string
	key        string
	logger     logging.Logger
	awsSession *session.Session
	goOutput   *s3.GetObjectOutput
}

func (obj *S3Object) Protocol() string {
	return protocolS3
}

func (obj *S3Object) Pretty() string {
	format := `S3Object { 
				region:  '%v' 
				endpoint: %v 
				bucket:  '%v' 
		        key :    '%v'
				logger:   %v 
				session: '%v'
			}`
	format = logging.Untabify(format, " ")
	args := logging.Prettify(obj.region, obj.formatEndpoint(), obj.bucket, obj.key, obj.logger, obj.formatSession())
	return fmt.Sprintf(format, args...)
}

func (obj *S3Object) String() string {
	return fmt.Sprintf(
		"{region: %v, endpoint: %v, bucket: %v, key: %v, logger: %v, awsSession: %v}",
		obj.region, obj.formatEndpoint(), obj.bucket, obj.key, obj.logger, obj.formatSession(),
	)
}

func (obj *S3Object) Reset() {
	obj.awsSession = nil
	obj.goOutput = nil
}

func (obj *S3Object) Logger() logging.Logger {
	return obj.logger
}

// Endpoint returns the endpoint URL used to access the object
func (obj *S3Object) Endpoint() *url.URL {
	return obj.endpoint
}

// Bucket returns the object's storage bucket
func (obj *S3Object) Bucket() *string {
	if obj.bucket == "" {
		return nil
	}
	return &obj.bucket
}

// Key returns the object's storage key
func (obj *S3Object) Key() *string {
	if obj.key == "" {
		return nil
	}
	return &obj.key
}

// ContentLength gets the size of the object in bytes, or returns an
// error if the size cannot be determined.
func (obj *S3Object) ContentLength() (int64, error) {
	goOutput, err := obj.getObject()
	if err != nil {
		obj.logger.Detailf("error determining content-length: %v\n", err)
		return 0, err
	}
	contentLength := goOutput.ContentLength
	if contentLength == nil {
		return 0, fmt.Errorf("no content-length returned by GetObject")
	}
	return *contentLength, nil
}

// SupportsRanges returns true if the object supports ranged downloads,
// false otherwise
func (obj *S3Object) SupportsRanges() bool {
	goOutput, err := obj.getObject()
	if err == nil {
		acceptRanges := goOutput.AcceptRanges
		if acceptRanges != nil {
			actual := *acceptRanges
			if "bytes" == actual {
				return true
			}
			obj.logger.Detailf("range request not supported; expected accept-ranges: 'bytes' but was '%v'\n", actual)
		} else {
			obj.logger.Detail("range request not supported; expected accept-ranges: 'bytes' but was no accept-ranges header found")
		}
	}
	return false
}

func (obj *S3Object) ReadRange(startInclusive, endInclusive int64, buffer []byte) (int64, error) {
	if !obj.SupportsRanges() {
		obj.logger.Detailf("object %v may not support ranged downloads; trying anyway\n", obj)
	}
	rangeStr := fmt.Sprintf("bytes=%d-%d", startInclusive, endInclusive)
	goInput := s3.GetObjectInput{
		Bucket: obj.Bucket(),
		Key:    obj.Key(),
		Range:  &rangeStr,
	}

	awsSession, err := obj.sessionP()
	if err != nil {
		return 0, err
	}
	downloader := s3manager.NewDownloader(awsSession)
	target := aws.NewWriteAtBuffer(buffer)
	return downloader.Download(target, &goInput)
}

func (obj *S3Object) StreamUp(body io.Reader, length int64) (err error) {
	awsSession, err := obj.sessionP()
	if err != nil {
		return err
	}
	// TODO: allow object to include an expected MD5
	uploader := s3manager.NewUploader(awsSession)
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: obj.Bucket(),
		Key:    obj.Key(),
		Body:   body,
	})
	if err == nil {
		obj.logger.Detailf("upload successful to %v\n", result.Location)
	}
	return err
}

func (obj *S3Object) Delete() (err error) {
	obj.Logger().Detailf("Delete: getting session for %v\n", ProtocolUriStr(obj))
	awsSession, err := obj.sessionP()
	if err != nil {
		return err
	}
	doInput := s3.DeleteObjectInput{
		Bucket: obj.Bucket(),
		Key:    obj.Key(),
	}
	obj.Logger().Detailf("deleting %v\n", ProtocolUriStr(obj))
	_, err = s3.New(awsSession).DeleteObject(&doInput)
	return err
}

// ------------------------------------------------------------
// Unexported functions

// Region returns the AWS region of the object
func (obj *S3Object) regionP() *string {
	if obj.region == "" {
		return nil
	}
	return &obj.region
}

func (obj *S3Object) sessionP() (*session.Session, error) {
	var err error
	if obj.awsSession == nil {
		endpointStr := obj.endpoint.String()
		obj.awsSession, err = protocols.ValidS3Session(&endpointStr, obj.regionP(), obj.logger)
	}
	return obj.awsSession, err
}

func (obj *S3Object) toGetObjectInput() *s3.GetObjectInput {
	goInput := s3.GetObjectInput{
		Bucket: obj.Bucket(),
		Key:    obj.Key(),
	}
	return &goInput
}

func (obj *S3Object) getObject() (*s3.GetObjectOutput, error) {
	var err error
	if obj.goOutput == nil {
		awsSession, err := obj.sessionP()
		if err == nil {
			s3Svc := s3.New(awsSession)
			goOutput, err := s3Svc.GetObject(obj.toGetObjectInput())
			if err != nil {
				return nil, err
			}
			if goOutput == nil {
				return nil, fmt.Errorf("nil *GetObjectOutput returned by S3.GetObject")
			}
			obj.goOutput = goOutput
		}
	}
	return obj.goOutput, err
}

func (obj *S3Object) formatSession() string {
	var sessionStr string
	if obj.awsSession == nil {
		sessionStr = "<nil>"
	} else {
		sessionStr = "(initialized)"
	}
	return sessionStr
}

func (obj *S3Object) formatEndpoint() string {
	var endpointStr string
	if obj.endpoint == nil {
		endpointStr = "<nil>"
	} else {
		endpointStr = obj.Endpoint().String()
	}
	return endpointStr
}
