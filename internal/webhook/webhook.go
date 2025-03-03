package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Panorama-Block/avax/internal/types"
)

type WebhookServer struct {
	EventChan chan types.WebhookEvent
	Port      string
}

func NewWebhookServer(eventChan chan types.WebhookEvent, port string) *WebhookServer {
	return &WebhookServer{
		EventChan: eventChan,
		Port:      port,
	}
}

func (ws *WebhookServer) Start() {
	http.HandleFunc("/webhook", ws.handleWebhook)

	log.Printf("Webhook server rodando na porta %s...", ws.Port)
	err := http.ListenAndServe(":"+ws.Port, nil)
	if err != nil {
		log.Fatalf("Erro ao iniciar webhook server: %v", err)
	}
}

func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler request", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var event types.WebhookEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	go func() {
		ws.EventChan <- event
	}()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Webhook recebido com sucesso")
}
