package main

import (
	"net/http"
	"protoapi"
	"protocore"
	"reflect"

	log "github.com/sirupsen/logrus"
)

type aProtobufWriter interface {
	WriteMessage(m *protoapi.Response) error
	WriteError(m *protoapi.Response, err error) error
}

type protobufHTTPWriter struct {
	writer http.ResponseWriter
	proto  *protocore.Proto
}

func newProtobufHTTPWriter(w http.ResponseWriter, proto *protocore.Proto) *protobufHTTPWriter {
	return &protobufHTTPWriter{
		writer: w,
		proto:  proto,
	}
}

func (w *protobufHTTPWriter) WriteMessage(m *protoapi.Response) error {
	w.writer.Header().Set("Content-Type", "application/octet-stream")
	w.writer.Header().Set("Cache-Control", "no-cache")

	return w.write(m)
}

func (w *protobufHTTPWriter) WriteError(m *protoapi.Response, err error) error {
	w.writer.Header().Set("Content-Type", "application/octet-stream")
	w.writer.Header().Set("Cache-Control", "no-cache")
	if linodeErr, ok := err.(*LinodeError); ok {
		if linodeErr.IsAuthError() {
			w.writer.WriteHeader(http.StatusUnauthorized)
		} else if linodeErr.IsPermissionsError() {
			w.writer.WriteHeader(http.StatusForbidden)
		} else {
			w.writer.WriteHeader(http.StatusTeapot)
		}
	} else {
		w.writer.WriteHeader(http.StatusTeapot)
	}

	return w.write(m)
}

func (w *protobufHTTPWriter) write(m *protoapi.Response) error {
	if err := w.proto.WriteMessage(w.writer, m); err != nil {
		log.WithFields(log.Fields{
			"cause":    err,
			"response": reflect.TypeOf(m.R).Name(),
		}).Error("Communication breakdown")
		return err
	}
	return nil
}
