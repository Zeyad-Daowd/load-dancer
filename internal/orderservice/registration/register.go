package registration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type RegisterBackendRequest struct {
	URL      string    `json:"url"`
	UniqueID uuid.UUID `json:"uniqueID"`
}

func attemptRegistration(fullURL string, requestBody []byte, controlPlaneSecret string, client *http.Client) (success bool, err error) {
	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(requestBody))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+controlPlaneSecret)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close() // closes as soon as this function returns

	return resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict, nil
}

func RegisterBackend(serverURL string, uniqueID uuid.UUID, controllerURL string, controlPlaneSecret string) error {

	reqPayload := RegisterBackendRequest{URL: serverURL, UniqueID: uniqueID}
	requestBody, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}
	fullURL := controllerURL + "/backends/register"
	maxRetries := 5
	retries := 0
	client := http.DefaultClient
	// Send the POST request to the controller
	for retries < maxRetries {
		success, err := attemptRegistration(fullURL, requestBody, controlPlaneSecret, client)
		if err != nil {
			slog.Error("Error sending registration request", "error", err)
		} else if success {
			slog.Info("Successfully registered backend server", "serverURL", serverURL, "controllerURL", controllerURL)
			return nil
		} else {
			slog.Warn("Failed to register backend server, will retry", "serverURL", serverURL, "controllerURL", controllerURL)
		}
		retries++
		time.Sleep(2 * time.Second) // wait before retrying
	}
	slog.Error("Failed to register backend server breaking out after max attempts", "serverURL", serverURL, "controllerURL", controllerURL)
	return fmt.Errorf("failed to register backend server after %d attempts", maxRetries)
}
func attemptDelete(client *http.Client, fullURL string, controlPlaneSecret string) (success bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fullURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Add("Authorization", "Bearer "+controlPlaneSecret)
	resp, err := client.Do(req)
	if err == nil && resp != nil && resp.StatusCode == http.StatusNotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

func DeregisterBackend(uniqueID uuid.UUID, controllerURL string, controlPlaneSecret string) error {
	fullURL, err := url.JoinPath(controllerURL, "backends", uniqueID.String())
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	maxRetries := 5
	retryDelay := 2 * time.Second
	client := http.DefaultClient

	for retries := 0; retries < maxRetries; retries++ {
		success, err := attemptDelete(client, fullURL, controlPlaneSecret)

		if err != nil {
			slog.Error("Error sending delete request", "error", err, "attempt", retries+1)
		} else if success {
			slog.Info("Successfully deleted backend server", "id", uniqueID, "controllerURL", controllerURL)
			return nil
		} else {
			slog.Warn("Failed to delete backend server, will retry", "id", uniqueID, "controllerURL", controllerURL)
		}
		time.Sleep(retryDelay)
	}

	slog.Error("Failed to delete backend server after max attempts", "id", uniqueID, "controllerURL", controllerURL)
	return fmt.Errorf("failed to delete backend server after %d attempts", maxRetries)
}
