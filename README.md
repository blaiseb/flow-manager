# Flow Manager

Un gestionnaire de demandes de flux réseau intelligent, conçu pour simplifier la saisie, le suivi et l'inventaire des ouvertures de ports entre machines ou sous-réseaux.

## 🚀 Fonctionnalités Clés

- **Architecture IP-Centric** : Les flux sont stockés par IP ou Subnet (CIDR), permettant de gérer aussi bien des machines isolées que des VLANs entiers.
- **Résolution Dynamique** : Les noms de machines (Hostname) et les noms de VLANs sont résolus en temps réel à l'affichage à partir des référentiels.
- **Référentiel VLAN avancé** :
    - Import/Export CSV (format : `subnet,name,gateway,dns`).
    - Vue détaillée par VLAN avec liste exhaustive de toutes les adresses IP du subnet.
    - Détection automatique de la Gateway et des serveurs DNS.
- **Gestion des CI (Configuration Items)** :
    - Autocomplétion intelligente lors de la saisie des demandes.
    - Création automatique des CI lors de la soumission de nouveaux flux.
- **Initialisation Automatique** : Chargement optionnel des référentiels (VLAN/CI) au premier démarrage via des fichiers CSV.
- **Workflow de Demande** :
    - Regroupement des lignes par **Référence unique** (`REF-YYYYMMDD-HHMMSS`).
    - **Génération automatique d'un fichier Excel** dès la validation de la demande.
    - Support des plages de ports (ex: `80, 443, 8080-8090`).
- **Consultation & Recherche** :
    - Recherche multicritère : par Hostname (via résolution inverse), par IP, par Subnet ou par commentaire.
    - Export complet de la base au format Excel.


## 🛠️ Stack Technique

- **Langage** : Go (Golang 1.26+)
- **Framework Web** : [Gin Gonic](https://github.com/gin-gonic/gin)
- **ORM** : [GORM](https://gorm.io/)
- **Sécurité** : CSRF Protection, OIDC (OpenID Connect), Bcrypt hashing
- **Base de données** : SQLite ou PostgreSQL
- **Interface** : HTML5, Bootstrap 5.3, JavaScript (Vanilla)
- **Excel** : [Excelize](https://github.com/xuri/excelize)

## 🔐 Sécurité & Configuration

L'application utilise des variables d'environnement pour la configuration sensible :

| Variable | Description | Par défaut |
| :--- | :--- | :--- |
| `FLOW_SESSION_SECRET` | Secret pour les sessions et CSRF (**Obligatoire en prod**) | `dev-secret-key` |
| `INITIAL_ADMIN_PASSWORD` | Mot de passe de l'admin par défaut au 1er boot | `admin` |
| `GIN_MODE` | Mode de fonctionnement de Gin (`release` ou `debug`) | `debug` |

### Authentification OIDC
Pour activer OIDC, configurez la section `auth.oidc` dans votre `config.yaml`. L'application gère automatiquement le rafraîchissement des jetons et la protection contre les attaques CSRF via un jeton `state` aléatoire.

## 🚀 Installation & Déploiement

### Installation Locale
1. Assurez-vous d'avoir Go installé (version 1.21+ recommandée, 1.26 testée).
2. Clonez le dépôt et installez les dépendances :
   ```bash
   go mod tidy
   ```
3. Testez l'application :
   ```bash
   go test ./...
   ```
4. Compilez et lancez :
   ```bash
   go build -o flow-manager main.go
   export FLOW_SESSION_SECRET="votre-secret-tres-long"
   ./flow-manager
   ```

### Déploiement Docker (Recommandé)
L'application est prête à être conteneurisée.

1. Préparez votre fichier `config.yaml` à partir de l'exemple.
2. Lancez avec Docker Compose :
   ```bash
   docker-compose up -d
   ```
   *Note : Les données (`flows.db`) sont persistées dans le dossier `./data` de votre hôte.*

## 🚦 Utilisation

1. Lancez l'application :
   ```bash
   ./flow-manager
   ```
   *Optionnel* :
   - Pour changer le port (8080 par défaut) : `./flow-manager -port 9090`
   - Pour activer le mode debug : `./flow-manager --debug`
2. Accédez à l'interface via votre navigateur : `http://localhost:8080`
3. **Initialisation** : Importez vos VLANs via l'onglet "VLANs > Importer CSV" pour activer la résolution dynamique, ou utilisez l'**Initialisation Automatique**.

### Initialisation Automatique (CSV)
Au premier démarrage, si la base de données est vide, l'application peut charger automatiquement vos référentiels depuis un dossier `init-data` à la racine :
- `init-data/vlans.csv` : pour initialiser les VLANs.
- `init-data/cis.csv` : pour charger le référentiel des serveurs.

**Exemple Docker Compose :**
```yaml
services:
  flow-manager:
    image: flow-manager:latest
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml
      - ./mes-referentiels:/app/init-data:ro
```

### Format d'import/export VLAN (CSV)
Le fichier doit être au format CSV avec des colonnes séparées par des **virgules** (`,`). 
Si plusieurs serveurs DNS sont renseignés, ils doivent être séparés par des **espaces** au sein de leur colonne :
```text
subnet,name,gateway,dns
192.168.1.0/24,VLAN_SERVERS,192.168.1.254,1.1.1.1 8.8.8.8
10.0.0.0/8,VLAN_CORP,10.0.0.1,10.0.0.2 10.0.0.3
```

### Format d'import CI (CSV)
Le fichier CSV pour les serveurs doit contenir au minimum les colonnes `ip` et `hostname`.
```text
ip,hostname,description
192.168.1.10,SRV-WEB-01,Serveur Web Production
192.168.1.20,SRV-DB-01,Base de données SQL
```

## 📂 Structure du Projet

- `main.go` : Point d'entrée, configuration des routes et du serveur.
- `handlers/` : Logique métier des différents modules (VLAN, CI, Flux, Export).
- `models/` : Définition des structures de données GORM.
- `database/` : Initialisation de SQLite et fonctions utilitaires réseau.
- `templates/` : Interface utilisateur (Template HTML unique avec JS intégré).

## 📝 Licence

Ce projet est sous licence MIT.
