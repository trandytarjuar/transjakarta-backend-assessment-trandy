package handler

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"strconv"
	"time"
	"transjakarta-fleet/internal/repository"
	"transjakarta-fleet/internal/service"
)

type Handler struct{ locations *service.LocationService }

func New(locations *service.LocationService) *Handler { return &Handler{locations} }
func (h *Handler) Register(app *fiber.App) {
	app.Get("/health", h.Health)
	app.Get("/vehicles/:vehicle_id/location", h.Latest)
	app.Get("/vehicles/:vehicle_id/history", h.History)
}
func (*Handler) Health(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}
func (h *Handler) Latest(c *fiber.Ctx) error {
	l, err := h.locations.Latest(c.UserContext(), c.Params("vehicle_id"))
	if errors.Is(err, repository.ErrNotFound) {
		return fail(c, 404, "not_found", "vehicle location not found")
	}
	if err != nil {
		return fail(c, 500, "internal_error", "could not retrieve location")
	}
	return c.JSON(fiber.Map{"data": l})
}
func (h *Handler) History(c *fiber.Ctx) error {
	startUnix, err := strconv.ParseInt(c.Query("start"), 10, 64)
	if err != nil || startUnix <= 0 {
		return fail(c, 400, "invalid_request", "start must be a positive Unix timestamp")
	}
	endUnix, err := strconv.ParseInt(c.Query("end"), 10, 64)
	if err != nil || endUnix <= 0 {
		return fail(c, 400, "invalid_request", "end must be a positive Unix timestamp")
	}
	start := time.Unix(startUnix, 0).UTC()
	end := time.Unix(endUnix, 0).UTC()
	items, err := h.locations.History(c.UserContext(), c.Params("vehicle_id"), start, end)
	if errors.Is(err, service.ErrInvalidLocation) {
		return fail(c, 400, "invalid_request", err.Error())
	}
	if err != nil {
		return fail(c, 500, "internal_error", "could not retrieve history")
	}
	return c.JSON(fiber.Map{"data": items, "count": len(items)})
}
func fail(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": code, "message": message}})
}
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return fail(c, code, "http_error", err.Error())
}
