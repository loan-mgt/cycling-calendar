package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var Log zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	Log = zerolog.New(os.Stdout).With().Timestamp().Logger()

	// file, err := os.OpenFile("/var/log/cyl-app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// if err != nil {
	// 	panic(err)
	// }
	// Log = zerolog.New(file).With().Timestamp().Logger()
}
