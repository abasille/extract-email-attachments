# Extract Email Attachments

Application Go pour extraire les pièces jointes des emails Gmail.

## Configuration

1. Créez un projet dans [Google Cloud Console](https://console.cloud.google.com)
2. Activez l'API Gmail
3. Configurez les identifiants OAuth 2.0 :
   - Type d'application : Application de bureau (Desktop app)
   - Ajoutez l'URL de redirection : `http://localhost:8080`
   - Téléchargez le fichier `client_secret.json` ou notez le Client ID et le Client Secret
4. Placez le client ID et le client secret dans la configuration de l'application (voir le code source)

## Authentification OAuth2 (PKCE)

- L'application utilise le flux OAuth2 avec PKCE, recommandé par Google pour les applications de bureau ([documentation officielle](https://developers.google.com/identity/protocols/oauth2/native-app?hl=fr#enable-apis)).
- Google requiert malgré tout un client secret pour les applications de bureau, même avec PKCE.
- Lors du premier lancement, une fenêtre de navigateur s'ouvre pour l'authentification et le consentement utilisateur.
- Le code d'autorisation est automatiquement récupéré via un serveur local (`http://localhost:8080`).
- Le token d'accès est stocké localement dans `token.json`.

## Installation

```bash
# Compilation
go mod download
go build

# Installation
./install.sh
```

## Utilisation

1. Lancez l'application
2. Suivez le processus d'authentification OAuth :
   - Une URL s'affichera dans la console (et/ou le navigateur s'ouvrira automatiquement)
   - Connectez-vous avec votre compte Google
   - Autorisez l'accès
   - Le code d'autorisation est récupéré automatiquement
3. Les pièces jointes seront extraites dans le dossier `attachments/`

## Sécurité

- Le Client ID et le Client Secret sont nécessaires pour l'authentification OAuth2
- Les tokens d'accès sont stockés localement dans `token.json`
- Utilisez des variables d'environnement ou un fichier de configuration pour les informations sensibles en production

## Développement

Pour le développement, vous pouvez utiliser votre propre Client ID et Client Secret en les modifiant dans le code source ou via un fichier de configuration.

## Licence

MIT 