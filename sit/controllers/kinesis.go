package controllers

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/pkg/errors"

	"github.com/lucidhq/go-sit/sit/events"
)

const partitionKey = "partition-key"

// StartLinearGeneratorFunc is function to launch linear event generator.
type StartLinearGeneratorFunc func(freq time.Duration, execFn events.ExecFunc) chan<- bool

// KinesisDataFunc is required to provide data for Kinesis record.
type KinesisDataFunc func() ([]byte, error)

// Kinesis is a controller to send events to the data stream.
type Kinesis struct {
	streamName    string
	kinesisClient kinesisiface.KinesisAPI
	stopChs       []chan<- bool
	errLogger     *log.Logger
}

// NewKinesis creates new Kinesis controller.
func NewKinesis(streamName string, kinesisClient kinesisiface.KinesisAPI) *Kinesis {
	return &Kinesis{
		streamName:    streamName,
		kinesisClient: kinesisClient,
		errLogger:     log.New(os.Stderr, "", 0),
	}
}

// MakeStartRecordsProducerFn creates function to start producer to put records
// to the Kinesis stream with certain frequency.
func (ctrl *Kinesis) MakeStartRecordsProducerFn(genratorFn StartLinearGeneratorFunc, dataFn KinesisDataFunc) func(freqStr string) error {
	recordPutFn := ctrl.MakeRecordPutFn(dataFn)

	return func(freqStr string) error {
		freq, err := time.ParseDuration(freqStr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse frequency %q for event generator", freqStr)
		}

		stopCh := genratorFn(freq, func() {
			if err := recordPutFn(); err != nil {
				ctrl.errLogger.Print(err)
			}
		})

		ctrl.stopChs = append(ctrl.stopChs, stopCh)

		return nil
	}
}

// MakeRecordPutFn creates function to put records to the Kinesis stream.
func (ctrl *Kinesis) MakeRecordPutFn(dataFn KinesisDataFunc) func() error {
	return func() error {
		data, err := dataFn()
		if err != nil {
			return errors.Wrapf(err, "failed to produce record data")
		}

		if _, err = ctrl.kinesisClient.PutRecord(&kinesis.PutRecordInput{
			Data:         data,
			PartitionKey: aws.String(partitionKey),
			StreamName:   &ctrl.streamName,
		}); err != nil {
			return errors.Wrapf(err, "failed to put record to stream %q", ctrl.streamName)
		}

		return nil
	}
}

// StopAllRecordsProducers stops all records producers.
func (ctrl *Kinesis) StopAllRecordsProducers() {
	for _, stopCh := range ctrl.stopChs {
		stopCh <- true
	}
	ctrl.stopChs = []chan<- bool{}
}

// WithErrLogger set custom logger for records producer errors.
func (ctrl *Kinesis) WithErrLogger(errLogger *log.Logger) *Kinesis {
	ctrl.errLogger = errLogger
	return ctrl
}
