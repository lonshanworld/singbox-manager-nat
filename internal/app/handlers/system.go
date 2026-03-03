package handlers

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"singbox-manager/internal/system"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

func (h *SystemHandler) PostRestartSingbox(c fiber.Ctx) error {
	log.Println("Manual restart of sing-box requested")
	if err := system.RestartSingbox(); err != nil {
		return c.Status(500).SendString("Failed to restart sing-box: " + err.Error())
	}
	return c.Redirect().To("/dashboard")
}

func (h *SystemHandler) PostStartSingbox(c fiber.Ctx) error {
	log.Println("Manual start of sing-box requested")
	if err := system.StartSingbox(); err != nil {
		return c.Status(500).SendString("Failed to start sing-box: " + err.Error())
	}
	return c.Redirect().To("/dashboard")
}

func (h *SystemHandler) PostStopSingbox(c fiber.Ctx) error {
	log.Println("Manual stop of sing-box requested")
	if err := system.StopSingbox(); err != nil {
		return c.Status(500).SendString("Failed to stop sing-box: " + err.Error())
	}
	return c.Redirect().To("/dashboard")
}
