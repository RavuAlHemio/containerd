//go:build windows

package wclayer

import (
	"context"
	"syscall"
	"time"

	"github.com/containerd/log"
	"github.com/Microsoft/hcsshim/internal/hcserror"
	"github.com/Microsoft/hcsshim/internal/oc"
	"go.opencensus.io/trace"
)

const (
	errnoERROR_SHARING_VIOLATION = 32
)

// ActivateLayer will find the layer with the given id and mount it's filesystem.
// For a read/write layer, the mounted filesystem will appear as a volume on the
// host, while a read-only layer is generally expected to be a no-op.
// An activated layer must later be deactivated via DeactivateLayer.
func ActivateLayer(ctx context.Context, path string) (err error) {
	title := "hcsshim::ActivateLayer"
	ctx, span := oc.StartSpan(ctx, title) //nolint:ineffassign,staticcheck
	defer span.End()
	defer func() { oc.SetSpanStatus(span, err) }()
	span.AddAttributes(trace.StringAttribute("path", path))

	logEntry := log.G(ctx)

	sleepSecs := 1
	for sleepSecs <= 30 {
		logEntry.Info("calling activateLayer")
		err = activateLayer(&stdDriverInfo, path)
		if err == nil {
			logEntry.Info("activateLayer succeeded")
			break
		}
		if errnoErr, ok := err.(syscall.Errno); ok {
			errnoInt := uintptr(errnoErr)
			logEntry.Infof("it's an errno error and our errno is %d", errnoInt)
			if errnoInt == errnoERROR_SHARING_VIOLATION {
				logEntry.Infof("it's a sharing violation; sleeping %d s this time", sleepSecs)
				time.Sleep(time.Duration(sleepSecs * 1000000000))
				sleepSecs = sleepSecs * 2
				continue
			}
		} else {
			logEntry.Info("it's not an errno error")
		}

		break
	}

	if err != nil {
		return hcserror.New(err, title, "")
	}
	return nil
}
