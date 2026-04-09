# Context Project: Flow Manager

Un gestionnaire intelligent de demandes de flux réseau, avec résolution dynamique et inventaire IP-centric.

## 🛠 Stack Technique
- **Backend** : Go 1.26+, Framework Gin Gonic.
- **Base de données** : GORM (SQLite par défaut, support PostgreSQL).
- **Frontend** : Template HTML5 unique (`index.html`), Bootstrap 5.3 (Internalisé), JavaScript Vanilla (`app.js`).
- **Sécurité** : Protection CSRF (`gin-csrf`), OIDC (Go-OIDC), Session cookies, Hachage Bcrypt.

## 🏗 Principes Architecturaux & Conventions
1. **IP-Centric** : Les flux sont stockés par IP ou Subnet (CIDR). Ne jamais remplacer l'IP par un hostname dans les schémas ; la résolution est dynamique et consultative via le référentiel CI.
2. **Ressources Embarquées (go:embed)** : Les fichiers statiques (`static/`) et templates (`templates/`) sont compilés dans le binaire (`main.go`) pour garantir une autonomie totale en environnement sans accès internet.
3. **Traçabilité Complète** : Chaque modification de flux doit impérativement générer une entrée dans `FlowHistory` et mettre à jour le champ `LastActor`. La date de modification (`ImplementedAt`) doit refléter la dernière action.
4. **Initialisation Automatique** : Le dossier `init-data/` sert au peuplement automatique des VLANs et CIs depuis des fichiers CSV au premier démarrage si les tables sont vides.
5. **Interopérabilité Template/JS** : L'utilisation de la fonction de template `JSONMarshal` permet l'injection de structures de données Go complexes vers le contexte JavaScript du frontend (ex: Historique).
6. **Frontend** : Le tri des tableaux est effectué côté client. Les colonnes ID techniques doivent être masquées dans l'interface au profit de liens contextuels basés sur les Hostnames ou VLANs.

## 📂 Structure Clé
- `/auth` : Logique d'authentification, middleware et mapping de permissions (Rôles: `viewer`, `requestor`, `actor`, `admin`).
- `/config` : Chargement de la configuration YAML (`config.yaml`) et des variables d'environnement (priorité).
- `/database` : Initialisation GORM, migration de schéma, algorithmes de résolution CI/VLAN optimisés et seeding initial (Admin par défaut).
- `/handlers` : Routage Gin (`routes.go`) et contrôleurs (API et Vues).
- `/models` : Schémas des entités GORM.
- `/static` & `/templates` : Ressources UI embarquées via `embed.FS`.

## 🚀 Commandes de Développement

### Installation
```bash
go mod tidy
```

### Build & Run
```bash
# Compilation avec CGO pour le support SQLite
CGO_ENABLED=1 GOOS=linux go build -o flow-manager main.go

# Lancement (avec secret requis en mode release)
export FLOW_SESSION_SECRET="votre-secret-de-session"
./flow-manager -port 8080 -debug
```

### Docker
L'application utilise un build multi-stage Docker. Les assets statiques étant embarqués, le conteneur final est minimaliste.
```bash
docker build -t flow-manager:latest .
docker-compose up -d
```

### Tests
```bash
go test ./...
```
