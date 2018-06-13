package main

import (
	"encoding/base64"
	"net/http"
	"strings"

	"protoapi"
	"protocore"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	log "github.com/sirupsen/logrus"
)

type protobufAPIServer struct {
	proto *protocore.Proto
}

func newProtobufAPIServer(hostKey []byte, peerKey []byte) *protobufAPIServer {
	return &protobufAPIServer{
		proto: protocore.NewProto(hostKey, peerKey),
	}
}

func (s *protobufAPIServer) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/*", s.handleVerb)
	return r
}

func (s *protobufAPIServer) handleVerb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	// Decode base64 payload.
	b64Data := strings.TrimSpace(chi.URLParam(r, "*"))
	if len(b64Data) == 0 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "empty verb", 400)
		return
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(b64Data)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "verb decode error: "+err.Error(), 400)
		return
	}

	// Decrypt message.
	request := &protoapi.Request{}
	err = s.proto.ReadMessage(request, ciphertext)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "verb decode error: "+err.Error(), 400)
		return
	}
	s.dispatchVerb(request, w, r)
}

func (s *protobufAPIServer) dispatchVerb(v *protoapi.Request, w http.ResponseWriter, r *http.Request) {
	writer := newProtobufHTTPWriter(w, s.proto)

	if args := v.GetLinodeCreateTunnel(); args != nil {
		s.logRequest(r, "Got request to create tunnel")
		newProtobufLinode(writer).CreateTunnel(args)
	} else if args := v.GetLinodeDestroyTunnel(); args != nil {
		s.logRequest(r, "Got request to destroy tunnel")
		newProtobufLinode(writer).DestroyTunnel(args)
	} else if args := v.GetLinodeRebuildTunnel(); args != nil {
		s.logRequest(r, "Got request to rebuild tunnel")
		newProtobufLinode(writer).RebuildTunnel(args)
	} else if args := v.GetLinodeTunnelStatus(); args != nil {
		s.logRequest(r, "Got request to retrieve tunnel status")
		newProtobufLinode(writer).TunnelStatus(args)
	} else if args := v.GetLinodeListInstances(); args != nil {
		s.logRequest(r, "Got request to list Linode instances")
		newProtobufLinode(writer).ListInstances(args)
	} else if args := v.GetLinodeListPlans(); args != nil {
		s.logRequest(r, "Got request to list Linode instance types")
		newProtobufLinode(writer).ListPlans(args)
	} else if args := v.GetLinodeListRegions(); args != nil {
		s.logRequest(r, "Got request to list Linode regions")
		newProtobufLinode(writer).ListRegions(args)
	} else if args := v.GetLinodeListImages(); args != nil {
		s.logRequest(r, "Got request to list Linode images")
		newProtobufLinode(writer).ListImages(args)
	} else if args := v.GetLinodeListStackscripts(); args != nil {
		s.logRequest(r, "Got request to list Linode StackScripts")
		newProtobufLinode(writer).ListStackScripts(args)
	} else {
		render.Status(r, 400)
		render.PlainText(w, r, "unsupported request")
	}
}

func (s *protobufAPIServer) logRequest(r *http.Request, msg string) {
	fields := log.Fields{
		"ip": r.RemoteAddr,
	}
	if h := r.Header.Get("X-Forwarded-For"); len(h) > 0 {
		fields["x-forwarded-for"] = h
	}
	if h := r.Header.Get("X-Real-IP"); len(h) > 0 {
		fields["x-real-ip"] = h
	}
	if h := r.Header.Get("CF-Connecting-IP"); len(h) > 0 {
		fields["cf-ip"] = h
	}
	if h := r.Header.Get("CF-IPCountry"); len(h) > 0 {
		fields["cf-country"] = h
	}
	log.WithFields(fields).Info(msg)
}
