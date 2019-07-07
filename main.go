package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	astronomer "github.com/ullaakut/astronomer/pkg/signature"
	"github.com/ullaakut/astronomer/pkg/trust"
)

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
	var report *astronomer.SignedReport

	data, err := ioutil.ReadFile(fmt.Sprintf("reports/%s-%s", repoOwner, repoName))
	if err != nil {
		return nil, fmt.Errorf("unable to read report: %v", err)
	}

	err = json.Unmarshal(data, &report)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal report: %v", err)
	}

	return report, nil
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var signedReport *astronomer.SignedReport
		err = json.Unmarshal(data, &signedReport)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		err = astronomer.Check(signedReport)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")

		log.Printf("Received valid report for %s/%s", signedReport.RepositoryOwner, signedReport.RepositoryName)

		err = storeReport(signedReport)
		if err != nil {
			log.Println("Unable to write report to filesystem:", err)
		}

		return
	})

	http.HandleFunc("/shields", func(w http.ResponseWriter, r *http.Request) {
		repoOwner := r.URL.Query().Get("owner")
		repoName := r.URL.Query().Get("name")

		badgeData := &astroBadge{
			SchemaVersion: 1,
			Label:         "astro rating",
		}

		log.Printf("Serving badge for %s/%s", repoOwner, repoName)

		report, err := fetchReport(repoOwner, repoName)
		if err != nil {
			badgeData.Color = "inactive"
			badgeData.Message = "unavailable"
		} else {
			if report.Factors[trust.Overall].TrustPercent >= 0.75 {
				badgeData.Color = "success"
			} else if report.Factors[trust.Overall].TrustPercent >= 0.5 {
				badgeData.Color = "yellow"
			} else {
				badgeData.Color = "red"
			}

			badgeData.Message = fmt.Sprintf("%1.f%%", report.Factors[trust.Overall].TrustPercent*100)
		}

		data, err := json.Marshal(badgeData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	log.Println("Listening on :80")

	log.Fatal(http.ListenAndServe(":80", nil))
}
