package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	broadcastPort = 9999
	tcpPort       = 8888
	broadcastAddr = "255.255.255.255"
	maxRetries    = 5
	retryDelay    = 2 * time.Second
)

var peers = make(map[string]bool)
var peersLock sync.Mutex

func main() {
	// Démarrer la diffusion
	go broadcastPresence()

	// Écouter la diffusion
	go listenForBroadcasts()

	// Démarrer le serveur TCP
	go startTCPServer()

	// Démarrer la connexion avec les pairs découverts
	connectToPeers()

	// Maintenir le programme en cours d'exécution
	select {}
}

func broadcastPresence() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastAddr, broadcastPort))
	if err != nil {
		fmt.Println("Erreur lors de la résolution de l'adresse UDP:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Erreur lors de la connexion UDP:", err)
		return
	}
	defer conn.Close()

	for {
		message := fmt.Sprintf("peer:%s", getLocalIP())
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi du message UDP:", err)
		}

		time.Sleep(5 * time.Second)
	}
}

func listenForBroadcasts() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", broadcastPort))
	if err != nil {
		fmt.Println("Erreur lors de la résolution de l'adresse UDP:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Erreur lors de l'écoute UDP:", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Erreur lors de la lecture UDP:", err)
			continue
		}

		message := string(buf[:n])
		if strings.HasPrefix(message, "peer:") {
			peerIP := strings.TrimPrefix(message, "peer:")
			peersLock.Lock()
			if peerIP != getLocalIP() {
				peers[peerIP] = true
				fmt.Println("Nouveau pair découvert:", peerIP)
			}
			peersLock.Unlock()
		}
	}
}

func startTCPServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", tcpPort))
	if err != nil {
		fmt.Println("Erreur lors de l'écoute TCP:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Serveur TCP démarré sur le port", tcpPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erreur lors de l'acceptation de la connexion TCP:", err)
			continue
		}

		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Erreur lors de la lecture TCP:", err)
		return
	}

	message := string(buf[:n])
	fmt.Println("Message reçu:", message)

	response := "Message reçu avec succès"
	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Println("Erreur lors de l'envoi de la réponse TCP:", err)
	}
}

func connectToPeers() {
	for {
		peersLock.Lock()
		for peer := range peers {
			go func(peer string) {
				err := retryableDial(peer)
				if err != nil {
					log.Println("Erreur lors de la connexion à", peer, ":", err)
				}
			}(peer)
		}
		peersLock.Unlock()
		time.Sleep(10 * time.Second)
	}
}

func connectToPeer(peer string) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", peer, tcpPort))
	if err != nil {
		fmt.Println("Erreur lors de la connexion à", peer, ":", err)
		return
	}
	defer conn.Close()

	message := "Hello from " + getLocalIP()
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Erreur lors de l'envoi du message TCP:", err)
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Erreur lors de la lecture de la réponse TCP:", err)
	}

	response := string(buf[:n])
	fmt.Println("Réponse de", peer, ":", response)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Erreur lors de la récupération des adresses réseau:", err)
		os.Exit(1)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return ""
}

func retryableDial(peer string) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", peer, tcpPort))
		if err == nil {
			defer conn.Close()
			message := "Hello from " + getLocalIP()
			_, err = conn.Write([]byte(message))
			if err != nil {
				log.Println("Erreur lors de l'envoi du message TCP:", err)
				return err
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				log.Println("Erreur lors de la lecture de la réponse TCP:", err)
				return err
			}

			response := string(buf[:n])
			log.Println("Réponse de", peer, ":", response)
			return nil
		}

		if !isRetryableError(err) {
			return err
		}

		log.Printf("Tentative %d: Échec de la connexion à %s, nouvelle tentative dans %s", attempt, peer, retryDelay)
		time.Sleep(retryDelay)
	}
	return errors.New("nombre maximum de tentatives atteint")
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	netErr, ok := err.(net.Error)
	if ok && netErr.Temporary() {
		return true
	}

	if strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "timeout") {
		return true
	}

	return false
}
