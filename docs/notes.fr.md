# Notes

Décisions, impasses et pièges, pour ne pas les payer deux fois.

## Où ça va

lumiases doit se placer dans un panneau à côté de Claude Code et le suivre en
direct : un hook de commande (`lumiases hook`) relaie chaque événement
(`PreToolUse`, `PostToolUse`, `Notification`, `Stop`...) en une ligne JSON vers
l'animateur, par un socket local, et sort aussitôt : un animateur fermé ne
ralentit jamais Claude. L'animateur traduit les événements en huit états et
suit le transcript de la session pour les phases qu'aucun hook ne voit
(réflexion, écriture). Rien de tout ça n'est encore construit : le rendu est
venu d'abord.

## Rendu

- **Braille et ASCII seulement.** Le « plasma » en demi-blocs, les anneaux
  topographiques et les metaballs ont été essayés puis écartés : à côté des
  deux autres, ils faisaient générique.
- **Braille : la densité porte la luminosité, la couleur garde sa teinte.**
  Les points sont fins ; moyenner les couleurs d'une cellule vers un bord
  sombre rendait la première orbe presque invisible. La couleur est désormais
  poussée à pleine intensité et c'est le nombre de points qui fait l'ombre.
- **Tramage braille : bruit à gradient entrelacé, pas Bayer.** Une matrice de
  Bayer 8×8 laissait voir une grille ; l'IGN répartit les points sans motif.
- **ASCII : caractère choisi par sa forme.** Chaque cellule est échantillonnée
  sur la même grille 2×4 que le braille et reçoit le caractère dont l'encre
  ressemble le plus ; les cellules unies prennent une gamme de densité, sinon
  toute zone grise se remplit de `|`. Les formes des caractères sont estimées
  à l'œil (`glyphShapes`), pas mesurées sur une police.
- **Pas de contour dessiné.** Deux contours ASCII ont été essayés (`| / - \`
  selon l'angle, puis `( ) _ ' .` placés là où le bord coupe la cellule) puis
  écartés : l'orbe se lit mieux avec un bord qui se fond dans le halo.
- **Pas d'épines sur error.** Des pointes sur le contour faisaient agressif ;
  error tremble désormais et les yeux se froncent.
- **Une goroutine par ligne.** Braille et ASCII prennent 8 échantillons par
  cellule, chacun avec plusieurs lectures de bruit ; réparti par lignes, un
  panneau de 200×60 tient environ 30 images par seconde.
- **Particules sans état.** L'âge d'une particule découle de la phase : rien
  n'est gardé d'une image à l'autre et elles suivent la vitesse de l'état.

## Mode statue

- **La photo ne bouge jamais ; c'est l'endroit où on la lit qui bouge.**
  Balancement, respiration, ondes, glitchs et fumée tordent tous les
  coordonnées avant la lecture : les effets se combinent d'eux-mêmes.
- **La transparence d'abord, la clé de couleur en secours.** L'alpha d'un PNG
  détouré sert de masque ; sans lui, « rouge au-dessus du vert » ne reconnaît
  que de la terre cuite sur du feuillage. Les photos de musée sont détourées
  avec Vision de macOS (le moteur de « Supprimer l'arrière-plan »), recadrées
  sur la tête et le cou, et rangées dans `heads/`, ignoré par git.
- **Rotation : un cylindre par ligne, l'arrière en miroir.** Le miroir fait se
  rejoindre les raccords à ±90° ; le profil reste vide (aucune photo ne l'a
  vu), d'où l'explosion douce à cet endroit. De vrais profils demandent des
  modèles 3D.
- **Un visage a besoin de contraste plus que de pixels.** À 30 colonnes par
  tête, les traits n'apparaissent qu'avec un masque flou fort et une courbe en
  S ; une police plus petite dans le panneau aide plus que n'importe quel
  réglage.
- **Masque flou au chargement.** Un visage a peu de cellules dans un
  terminal ; sans pousser le contraste local, les yeux, le nez et la bouche
  fondent dans les joues.
- **Aucune photo dans le dépôt.** Celle utilisée pendant la mise au point vient
  d'une banque d'images ; vérifier la licence d'une image avant d'en livrer une.

## Pièges

- Captures : `COLUMNS=110 LINES=50 go run . -frames 45 < /dev/null > out.ansi`.
  Sans terminal, la taille vient de `COLUMNS` et `LINES`, et les touches sont
  coupées. Les images sont séparées par `ESC[?2026h`.
- `eyeStyle` et `face` sont des globales lues par les goroutines de lignes :
  ne les changer qu'entre deux images, comme le fait `main`.
- Des caractères plus petits ne se demandent pas depuis le programme : c'est la
  taille de police du panneau. Certains terminaux la changent par panneau,
  d'autres par fenêtre.

## Vérifié sur

| Système | Quoi |
|---|---|
| macOS (Darwin 27), Apple Silicon | Compile, tests verts, chaque style et le mode statue regardés dans un terminal, ~30 i/s à 200×60 ; le style image regardé dans Ghostty seulement (~24 i/s à 100×46) |
| Linux | `GOOS=linux go vet` seulement ; jamais lancé |
| Windows | `GOOS=windows go vet` seulement ; jamais lancé. Pas de taille de terminal (repli sur `COLUMNS`/`LINES`), pas de touches (pas de `stty`) |
