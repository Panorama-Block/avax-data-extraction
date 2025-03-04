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

	log.Printf("Servidor Webhook rodando na porta %s...", ws.Port)
	if err := http.ListenAndServe(":"+ws.Port, nil); err != nil {
		log.Fatalf("Erro ao iniciar o servidor Webhook: %v", err)
	}
}

func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler a requisição", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var event types.WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	go func() {
		ws.EventChan <- event
	}()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Webhook recebido com sucesso")
}
