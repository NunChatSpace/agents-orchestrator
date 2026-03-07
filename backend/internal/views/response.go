package views

import (
	"encoding/json"
	"net/http"
)

type Meta struct {
	RequestID string `json:"request_id,omitempty"`
}

type envelope struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error errorDetail `json:"error"`
	Meta  Meta        `json:"meta"`
}

func JSON(w http.ResponseWriter, status int, data interface{}, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{Data: data, Meta: Meta{RequestID: requestID}})
}

type metaWithTotal struct {
	RequestID  string `json:"request_id,omitempty"`
	TotalItems int    `json:"total_items"`
}

type envelopeWithTotal struct {
	Data interface{}    `json:"data"`
	Meta metaWithTotal  `json:"meta"`
}

func JSONWithTotal(w http.ResponseWriter, status int, data interface{}, total int, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelopeWithTotal{
		Data: data,
		Meta: metaWithTotal{RequestID: requestID, TotalItems: total},
	})
}

func Error(w http.ResponseWriter, status int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorEnvelope{
		Error: errorDetail{Code: code, Message: message},
		Meta:  Meta{RequestID: requestID},
	})
}
