# Extract Email Attachments

## Description

Cette application en Go permet d'extraire automatiquement toutes les pièces jointes PDF des emails reçus sur une boîte Gmail, depuis la dernière exécution.
Les fichiers sont téléchargés dans le sous-dossier `extract-email-attachments` de vos téléchargements, avec gestion de l'historique pour éviter de télécharger plusieurs fois le même document.
L'authentification s'effectue via OAuth2 (PKCE) et aucune donnée n'est transmise à un service tiers autre que Google Gmail (accès en lecture seule).

## Objectifs du projet

### Pour 2025

- Récupérer toutes les factures envoyées en pièce-jointe d'email, les renommer en respectant un format défini (ex : YYYY-MM-facture-ENTREPRISE-ID) et les déplacer dans un des dossiers stockant mes factures.
- Utiliser un OCR IA (par exemple Mistral OCR) pour récupérer le nom de l'entreprise, la date d'émission de la facture et le n° de la facture lorsque ces informations ne peuvent pas être déduites de l'email ou du nom de fichier de la pièce-jointe.

### À terme

Analyser la structure des dossiers de mon disque dur, la nature de leur contenu (factures personnelles, devis du conseil syndical, attestation fiscale) et les schémas de nommage des fichiers, pour en déduire comment une pièce-jointe doit être renommée et dans quel dossier la télécharger.

## Prérequis techniques

- Go 1.20 ou supérieur.
- OS : Testé uniquement avec macOS. Devrait fonctionner sous Linux moyennant quelques adpatations.
- Seul Gmail est supporté actuellement.

Application Go pour extraire les pièces jointes des emails Gmail.

## Configuration

1. Créez un projet dans [Google Cloud Console](https://console.cloud.google.com)
2. Activez l'API Gmail
3. Configurez les identifiants OAuth 2.0 :
   - Type d'application : Application de bureau (Desktop app)
   - Ajoutez l'URL de redirection : `http://localhost:8080`
   - Téléchargez le fichier `client_secret.json` dans `./config/extract-email-attachments` ou renseignez les variables d'environnement `GOOGLE_CLIENT_ID` et `GOOGLE_CLIENT_SECRET`.

## Authentification OAuth2 (PKCE)

- L'application utilise le flux OAuth2 avec PKCE, recommandé par Google pour les applications de bureau ([documentation officielle](https://developers.google.com/identity/protocols/oauth2/native-app?hl=fr#enable-apis)).
- Google requiert malgré tout un client secret pour les applications de bureau, même avec PKCE.
- Lors du premier lancement, une fenêtre de navigateur s'ouvre pour l'authentification et le consentement utilisateur.
- Le code d'autorisation est automatiquement récupéré via un serveur local (`http://localhost:8080`).
- Le token d'accès est stocké localement dans `./config/extract-email-attachments/caches/token.json`.

## Installation

```bash
# Compilation
go mod download
go build

# Installation
./install.sh
```

Ajouter `~/.bin/` à votre `$PATH`.

## Utilisation

1. Lancez l'application `extract-email-attachments`
2. Suivez le processus d'authentification OAuth :
   - Le navigateur par défaut s'ouvrira automatiquement
   - Connectez-vous avec votre compte Google
   - Autorisez l'accès
   - Le code d'autorisation est récupéré automatiquement
3. Les pièces jointes seront extraites dans le sous-dossier `attachments/` des téléchargements.

## Tests

Pour exécuter les tests :

```bash
# Exécuter tous les tests
go test ./...

# Exécuter les tests avec plus de détails
go test -v ./...

# Voir la couverture de test
go test -cover ./...
```

## Licence

MIT License

Copyright (©) 2025 Aurélien BASILLE
