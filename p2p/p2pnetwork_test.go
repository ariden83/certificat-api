package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"testing"
	"time"
)

// TestBroadcastPresence vérifie si la fonction broadcastPresence envoie correctement
// des messages de présence via UDP.
func TestBroadcastPresence(t *testing.T) {
	// Créer un buffer pour capturer les messages envoyés via UDP
	var receivedMessage []byte
	var receivedAddr *net.UDPAddr

	// Créer un faux serveur UDP pour écouter les messages envoyés par broadcastPresence
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastAddr, broadcastPort))
	if err != nil {
		t.Fatalf("Erreur lors de la résolution de l'adresse UDP: %v", err)
	}
	conn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Erreur lors de l'écoute UDP: %v", err)
	}
	defer conn.Close()

	// Lancer broadcastPresence dans une goroutine
	done := make(chan struct{})
	go func() {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastAddr, broadcastPort))
		if err != nil {
			log.Println("Erreur lors de la résolution de l'adresse UDP:", err)
			return
		}

		// Connexion UDP pour l'envoi de messages de diffusion
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Println("Erreur lors de la connexion UDP:", err)
			return
		}
		defer conn.Close()

		// Construction du message de présence contenant l'adresse IP locale
		message := fmt.Sprintf("peer:%s", getLocalIP())

		// Envoi du message de présence via la connexion UDP
		if _, err := conn.Write([]byte(message)); err != nil {
			log.Println("Erreur lors de l'envoi du message UDP:", err)
		}
		close(done)
	}()

	// Attendre que le message soit envoyé et capturé par le faux serveur UDP
	time.Sleep(1 * time.Second)

	// Lire le message reçu par le faux serveur UDP
	buffer := make([]byte, 1024)
	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Erreur lors de la lecture du message UDP: %v", err)
	}
	receivedMessage = buffer[:n]
	receivedAddr = addr

	// Vérifier si le message est correct
	expectedMessage := []byte("peer:" + getLocalIP())
	if !bytes.Equal(receivedMessage, expectedMessage) {
		t.Errorf("Message reçu incorrect: got %s, want %s", receivedMessage, expectedMessage)
	}
	// Vérifier si l'adresse du destinataire est correcte
	expectedAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastAddr, broadcastPort))
	if err != nil {
		t.Fatalf("Erreur lors de la résolution de l'adresse UDP: %v", err)
	}

	if serverAddr.IP.String() != expectedAddr.IP.String() || serverAddr.Port != expectedAddr.Port {
		t.Errorf("Adresse du destinataire incorrecte: got %s, want %s", addr, expectedAddr)
	}
	if serverAddr.String() != expectedAddr.String() {
		t.Errorf("Adresse du destinataire incorrecte: got %s, want %s", receivedAddr, expectedAddr)
	}

	// Arrêter la goroutine une fois que le message a été vérifié
	<-done
}
