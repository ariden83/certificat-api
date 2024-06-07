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
	// broadcastPort est le numéro de port sur lequel les messages UDP seront envoyés pour la diffusion.
	// C'est le port sur lequel les autres pairs écoutent pour détecter les nouveaux pairs sur le réseau.
	// Dans cet exemple, il est défini comme 9999.
	broadcastPort = 9999
	tcpPort       = 8888
	// broadcastAddr est l'adresse IP de diffusion, c'est-à-dire l'adresse à laquelle les messages UDP
	// seront envoyés pour être diffusés à tous les pairs connectés au même réseau local.
	// Dans cet exemple, il est défini comme 255.255.255.255, qui est l'adresse IP de diffusion par défaut
	// pour les réseaux IPv4.
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

// broadcastPresence diffuse périodiquement la présence du pair sur le réseau en utilisant UDP.
// Il envoie un message contenant l'adresse IP locale du pair sur l'adresse de diffusion spécifiée.
// Cela permet aux autres pairs de découvrir et d'ajouter ce pair à leur liste de pairs actifs.
func broadcastPresence() {
	// Résolution de l'adresse UDP pour la diffusion sur le port spécifié
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

	// Boucle pour envoyer périodiquement des messages de présence
	for {
		// Construction du message de présence contenant l'adresse IP locale
		message := fmt.Sprintf("peer:%s", getLocalIP())

		// Envoi du message de présence via la connexion UDP
		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Println("Erreur lors de l'envoi du message UDP:", err)
		}

		// Attente avant d'envoyer le prochain message de présence
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
