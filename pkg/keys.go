package pkg

import (
	"github.com/dmolesUC3/cos/internal/keys"
	"github.com/dmolesUC3/cos/internal/logging"
)

type Keys struct {
	endpoint string
	region   string
	bucket   string
	logger   logging.Logger
}

func NewKeys(endpoint, region, bucket string, logger logging.Logger) Keys {
	return Keys{endpoint, region, bucket, logger}
}

type KeyFailure struct {
	Source string
	Index  int
	Key    string
	Error  error
}

func (k *Keys) CheckAll(source keys.Source) ([]KeyFailure, error) {
	var failures []KeyFailure
	for index, key := range source.Keys() {
		f, err := k.Check(source.Name(), index, source.Count(), key)
		if err != nil {
			return nil, err
		}
		if f != nil {
			failures = append(failures, *f)
		}
	}
	return failures, nil
}

func (k *Keys) Check(sourceName string, index, count int, key string) (*KeyFailure, error) {
	crvd, err := NewDefaultCrvd(key, k.endpoint, k.region, k.bucket, k.logger)
	if err != nil {
		return nil, err
	}
	k.logger.Infof("%d of %d from %v\n", 1 + index, count, sourceName)
	err = crvd.CreateRetrieveVerifyDelete()
	if err != nil {
		k.logger.Infof("%#v (%d of %d from %v) failed: %v\n", key, 1 + index, count, sourceName, err)
		return &KeyFailure{sourceName, index, key, err}, nil
	}
	return nil, err
}
