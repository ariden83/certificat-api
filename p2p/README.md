# Peer to peer

## Functioning

### 1. Diffusion de la Présence:

- Chaque machine envoie régulièrement un message UDP en diffusant son adresse IP.
- Cela permet aux autres machines d’apprendre l'existence de cette machine.

### 2. Écoute des Diffusions :

- Chaque machine écoute les messages de diffusion sur le port UDP pour découvrir de nouveaux pairs.
- Lorsqu'un nouveau pair est détecté, son adresse IP est ajoutée à une liste de pairs connus.

### 3. Serveur TCP :

- Chaque machine démarre un serveur TCP pour écouter les connexions entrantes des autres pairs.

### 4. Connexion aux Pairs :

- Chaque machine tente régulièrement de se connecter aux pairs connus via TCP pour envoyer un message et recevoir une réponse.

### 5. Récupération de l'Adresse IP Locale :

- La fonction `getLocalIP` permet de récupérer l'adresse IP locale de la machine pour la diffusion.

### 6. Reconnexion en cas d'échec de connexion :

- La fonction `retryableDial` tente de se connecter au pair spécifié jusqu'à un maximum de maxRetries.
- Si la connexion échoue, la fonction attend `retryDelay` avant de réessayer.

### 7. Vérification des erreurs récupérables :

- La fonction `isRetryableError` vérifie si une erreur est temporaire ou liée à la connectivité réseau, ce qui signifie qu'une reconnexion peut réussir.
- Les erreurs comme connection refused ou timeout sont considérées comme récupérables.

### 8. Boucle de reconnexion :

- `retryableDial` utilise une boucle pour essayer de se connecter jusqu'à maxRetries fois.
- En cas d'échec à chaque tentative, la fonction attend `retryDelay` avant de réessayer.

### 9. Logs d'erreurs et gestion des erreurs :

- Les logs fournissent des informations sur l’état des tentatives de connexion, ce qui est utile pour le diagnostic.
- Si la connexion échoue après toutes les tentatives, une erreur est retournée.

### 10. Déconnexion automatique :

- Les connexions TCP sont automatiquement fermées à la fin de la fonction `retryableDial` grâce à defer conn.Close(), garantissant que les ressources sont libérées correctement.

## Exécution du Programme

### 1. Configuration des Variables d'Environnement :

Assurez-vous que le programme a les permissions nécessaires pour utiliser les ports UDP et TCP.

### 2. Installation

Command to generate the binary :

> make build

Command to generate the binary and run it :

> make run

### 3. Observation :

Les machines vont commencer à découvrir leurs pairs et à communiquer entre elles via TCP.
