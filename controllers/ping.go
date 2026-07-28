package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PingController struct{}

func NewPingController() *PingController {
	return &PingController{}
}

func (pc *PingController) Ping(c *gin.Context) {
	Response(c, http.StatusOK, "pong", nil)
}