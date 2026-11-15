package controller

import (
	"encoding/json"
	"net/http"
)

type DumbServerController struct{}

func NewDumbServer() *DumbServerController {
	return &DumbServerController{}
}

func (c *DumbServerController) DumbServerController(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}