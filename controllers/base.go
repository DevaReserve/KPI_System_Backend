package controllers

import (
	"KPI_System_Backend/global_var"

	"github.com/gin-gonic/gin"
)

func Response(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, global_var.ResponseFormat{
		Status:  status,
		Message: message,
		Data:    data,
	})
}