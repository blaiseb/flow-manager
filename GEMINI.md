# Context Project: Flow Manager

Gestionnaire intelligent de demandes de flux réseau avec résolution dynamique et inventaire IP-centric.

## 🛠 Stack Technique
- **Backend** : Go 1.26+, Framework Gin Gonic.
- **Base de données** : GORM (SQLite par défaut, support PostgreSQL).
- **Frontend** : Template unique (`index.html`), Bootstrap 5.3, JavaScript Vanilla (`app.js`).
- **Sécurité** : Protection CSRF (`gin-csrf`), OIDC (Go-OIDC), Hachage Bcrypt.

## 🏗 Principes Architecturaux
1. **IP-Centric** : Les flux sont stockés par IP ou Subnet (CIDR). Ne jamais supprimer l'IP au profit d'un nom de machine ; le nom de machine est une donnée consultative issue du référentiel CI.
2. **Résolution Dynamique** : Les Hostnames et noms de VLAN sont résolus à la volée lors de l'affichage.
3. **Traçabilité** : Chaque modification de flux doit générer une entrée dans `FlowHistory` et mettre à jour `LastActor`.
4. **Initialisation** : Le dossier `init-data/` est utilisé pour le peuplement automatique des VLANs/CIs au premier démarrage si les tables sont vides.

## 📜 Conventions de Code
- **Go** : Suivre les idiomes standards. Les handlers sont regroupés par domaine dans `handlers/`.
- **Modèles** : Tous les modèles GORM résident dans `models/models.go`.
- **Frontend** : 
    - Le passage de données complexes de Go vers le JavaScript du template doit utiliser la fonction `JSONMarshal`.
    - Maintenir la logique de pré-remplissage des flux via la table `StandardFlow`.
- **Commits** : Utiliser le format **Conventional Commits** (ex: `feat:`, `fix:`, `refactor:`, `docs:`).

## 🔒 Sécurité
- Toute nouvelle route protégée doit utiliser le middleware `auth.AuthRequired`.
- Ne jamais désactiver la vérification CSRF, sauf pour les routes de callback OIDC si nécessaire.
- Le secret de session (`FLOW_SESSION_SECRET`) doit être lu en priorité depuis l'environnement.

## 📂 Structure Clé
- `/handlers` : Logique métier et API.
- `/models` : Définition des schémas.
- `/database` : Connexion, migration et seeding.
- `/static/js/app.js` : Toute la logique dynamique frontend.
- `/templates/index.html` : Interface principale.

## 🚀 Commandes utiles
- Build : `CGO_ENABLED=1 go build -o flow-manager main.go`
- Test : `go test ./...`
- Debug : `./flow-manager --debug`
