package controllers_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/Fellah/go-sit/sit/sit/controllers"
	"github.com/Fellah/go-sit/sit/sit/controllers/mocks"
	"github.com/Fellah/go-sit/sit/sit/events"
)

//go:generate mockgen -package=mocks -destination=./mocks/kinesisapi.go github.com/aws/aws-sdk-go/service/kinesis/kinesisiface KinesisAPI

type RecordMatcher struct {
	Data []byte
}

func (m RecordMatcher) Matches(x interface{}) bool {
	return bytes.Equal(m.Data, x.(*kinesis.PutRecordInput).Data)
}

func (m RecordMatcher) String() string {
	return fmt.Sprintf("records' data doesn't match %q", (m.Data))
}

func TestKinesisRecordsProducer(t *testing.T) {
	tCases := map[string]struct {
		freq             string
		dataFn           controllers.KinesisDataFunc
		mockKinesisAPIFn func(*mocks.MockKinesisAPI)
		eventGeneratorFn controllers.StartLinearGeneratorFunc
		errorAssertionFn assert.ErrorAssertionFunc
	}{
		"OK": {
			freq: "1s",
			dataFn: func() ([]byte, error) {
				return []byte("some-message"), nil
			},
			errorAssertionFn: assert.NoError,
			mockKinesisAPIFn: func(mk *mocks.MockKinesisAPI) {
				mk.
					EXPECT().
					PutRecord(RecordMatcher{Data: []byte("some-message")}).
					Return(&kinesis.PutRecordOutput{}, nil).
					Times(1)
			},
			eventGeneratorFn: func(freq time.Duration, execFn events.ExecFunc) chan<- bool {
				assert.Equal(t, time.Second, freq)
				execFn()
				return make(chan bool)
			},
		},

		"IncorrectFrequency": {
			freq:   "one-second",
			dataFn: func() ([]byte, error) { return nil, nil },
			mockKinesisAPIFn: func(mk *mocks.MockKinesisAPI) {
				mk.EXPECT().PutRecord(gomock.Any()).Times(0)
			},
			eventGeneratorFn: func(freq time.Duration, execFn events.ExecFunc) chan<- bool {
				assert.Fail(t, "event generator shouldn't start")
				return make(chan bool)
			},
			errorAssertionFn: assert.Error,
		},
	}

	for tn, tc := range tCases {
		t.Run(tn, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			kinesisAPIMock := mocks.NewMockKinesisAPI(ctrl)
			tc.mockKinesisAPIFn(kinesisAPIMock)

			kinesisCtrl := controllers.NewKinesis("stream-name", kinesisAPIMock)

			startRecordsProducerFn := kinesisCtrl.MakeStartRecordsProducerFn(tc.eventGeneratorFn, tc.dataFn)
			tc.errorAssertionFn(t, startRecordsProducerFn(tc.freq))
		})
	}
}
