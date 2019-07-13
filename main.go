package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	echo "github.com/labstack/echo/v4"
	middleware "github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	astronomer "github.com/ullaakut/astronomer/pkg/signature"
	"github.com/ullaakut/astronomer/pkg/trust"
)

var log *zerolog.Logger

type astroBadge struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

func storeReport(report *astronomer.SignedReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("unable to marshal report: %v", err)
	}

	err = ioutil.WriteFile(fmt.Sprintf("reports/%s-%s", report.RepositoryOwner, report.RepositoryName), data, 0655)
	if err != nil {
		return err
	}

	return nil
}

func fetchReport(repoOwner, repoName string) (*astronomer.SignedReport, error) {
	report := &astronomer.SignedReport{}

	data, err := ioutil.ReadFile(fmt.Sprintf("reports/%s-%s", repoOwner, repoName))
	if err != nil {
		return nil, fmt.Errorf("unable to read report: %v", err)
	}

	err = json.Unmarshal(data, report)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal report: %v", err)
	}

	return report, nil
}

func percentToLetterGrade(percent float64) string {
	switch {
	case percent > 0.8:
		return "A"
	case percent > 0.6:
		return "B"
	case percent > 0.4:
		return "C"
	case percent > 0.2:
		return "D"
	default:
		return "E"
	}
}

func handleReport(ctx echo.Context) error {
	var signedReport astronomer.SignedReport
	err := ctx.Bind(&signedReport)
	if err != nil {
		err = errors.Wrap(err, "could not parse blog post from request body")
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = astronomer.Check(&signedReport)
	if err != nil {
		err = errors.Wrap(err, "invalid signature for report")
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	err = storeReport(&signedReport)
	if err != nil {
		err = errors.Wrap(err, "unable to write report to filesystem")
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusCreated, signedReport)
}

func handleBadge(ctx echo.Context) error {
	repoOwner := ctx.QueryParam("owner")
	if len(repoOwner) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "owner not set in request context")
	}

	repoName := ctx.QueryParam("name")
	if len(repoName) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "repository name not set in request context")
	}

	badgeData := &astroBadge{
		SchemaVersion: 1,
		Label:         "astro rating",
	}

	report, err := fetchReport(repoOwner, repoName)
	if err != nil {
		badgeData.Color = "inactive"
		badgeData.Message = "unavailable"
	} else {
		if report.Factors[trust.Overall].TrustPercent > 0.6 {
			badgeData.Color = "success"
		} else if report.Factors[trust.Overall].TrustPercent > 0.4 {
			badgeData.Color = "yellow"
		} else {
			badgeData.Color = "red"
		}

		badgeData.Message = percentToLetterGrade(report.Factors[trust.Overall].TrustPercent)
	}

	return ctx.JSON(http.StatusOK, badgeData)
}

func main() {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	// Use zerolog for debugging HTTP requests
	log = NewZeroLog(os.Stderr)
	e.Logger.SetLevel(5) // Disable default logging
	e.Use(HTTPLogger(log))

	e.POST("/", handleReport)
	e.GET("/shields", handleBadge)

	e.Logger.Fatal(e.Start("0.0.0.0:4242"))
}
