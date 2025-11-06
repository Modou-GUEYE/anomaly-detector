# Fichiers de Test pour l'Import de Pays à Risque

Ce dossier contient des fichiers d'exemple pour tester la fonctionnalité d'import de pays à risque dans différents formats.

## Fichiers disponibles :

### 1. **pays-risque.csv**
- Format : CSV avec en-tête
- Structure :
  ```
  nom_pays
  Afghanistan
  Albanie
  ...
  ```
- Contient : 16 pays

### 2. **pays-risque.json** 
- Format : JSON avec objet
- Structure :
  ```json
  {
    "pays": ["Afghanistan", "Albanie", ...]
  }
  ```
- Contient : 16 pays

### 3. **pays-risque-array.json**
- Format : JSON tableau simple
- Structure :
  ```json
  ["Biélorussie", "Erythrée", ...]
  ```
- Contient : 10 pays

### 4. **pays-risque.xlsx**
- Format : Excel avec en-tête
- Structure : Colonne "nom_pays" avec les noms de pays
- Contient : 16 pays

### 5. **pays-risque-sans-entete.xlsx**
- Format : Excel sans en-tête
- Structure : Première colonne contient directement les noms de pays
- Contient : 10 pays

## Comment tester :

1. Accédez à la page "Pays à Risque" dans l'application
2. Cliquez sur "📥 Importer"
3. Sélectionnez l'un des fichiers de test
4. Cliquez sur "Importer"
5. Vérifiez que les pays sont correctement importés

## Notes importantes :

- Les fichiers Excel sont traités côté client (JavaScript)
- Les fichiers CSV/JSON sont traités côté serveur
- Le système détecte automatiquement les en-têtes dans les fichiers Excel
- Les doublons sont automatiquement évités

## Formats supportés :

- ✅ CSV (.csv)
- ✅ JSON (.json)
- ✅ Excel (.xlsx, .xls)
- ✅ Fichiers texte (.txt)