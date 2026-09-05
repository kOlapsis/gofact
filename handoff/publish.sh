#!/usr/bin/env bash
# publish.sh — publie un contenu de handoff/comms/queue/ sur le canal indiqué par son
# en-tête YAML.
#
#   ./handoff/publish.sh handoff/comms/queue/2026-09-08-linkedin-1er-septembre.md
#   ./handoff/publish.sh --dry-run <fichier>     # n'appelle aucune API
#
# Codes de sortie
#   0  publié (ou simulation réussie en --dry-run)
#   1  erreur d'usage ou fichier invalide
#   2  l'API a refusé la publication
#   3  jeton absent pour ce canal — repli manuel attendu
#
# Jetons lus dans l'environnement, jamais dans le dépôt :
#   LINKEDIN_ACCESS_TOKEN   jeton OAuth 2.0, portée w_member_social
#   LINKEDIN_AUTHOR_URN     urn:li:person:XXXXXXXX
#   LINKEDIN_API_VERSION    optionnel, défaut 202608
#   X_ACCESS_TOKEN          jeton OAuth 2.0 utilisateur, portée tweet.write

set -euo pipefail

DRY_RUN=0
if [ "${1:-}" = "--dry-run" ]; then DRY_RUN=1; shift; fi

FILE="${1:-}"
if [ -z "$FILE" ] || [ ! -f "$FILE" ]; then
  echo "usage: $0 [--dry-run] <fichier de comms/queue>" >&2
  exit 1
fi

# --- en-tête ---------------------------------------------------------------
field() { sed -n '2,/^---$/p' "$FILE" | grep -m1 "^$1:" | sed "s/^$1:[[:space:]]*//" | sed 's/^"//; s/"$//'; }

CANAL="$(field canal)"
STATUT="$(field statut)"
BODY="$(sed '1{/^---$/!q}; 1,/^---$/d' "$FILE" | sed '/./,$!d')"

if [ -z "$CANAL" ]; then echo "en-tête sans champ 'canal' : $FILE" >&2; exit 1; fi
if [ -z "$BODY" ];  then echo "corps vide : $FILE" >&2; exit 1; fi
if [ "$STATUT" = "publié" ]; then echo "déjà publié, rien à faire : $FILE" >&2; exit 0; fi

echo "canal=$CANAL  signes=${#BODY}  fichier=$FILE" >&2

# --- garde-fous ------------------------------------------------------------
case "$CANAL" in
  linkedin)
    if [ "${#BODY}" -gt 2900 ]; then echo "post LinkedIn trop long (${#BODY} signes, max 2900)" >&2; exit 1; fi ;;
  x)
    if [ "${#BODY}" -gt 280 ]; then echo "post X trop long (${#BODY} signes, max 280)" >&2; exit 1; fi ;;
esac

if [ "$DRY_RUN" = "1" ]; then
  echo "--- simulation, aucun appel réseau ---" >&2
  printf '%s\n' "$BODY"
  exit 0
fi

# --- canaux ----------------------------------------------------------------
case "$CANAL" in

  linkedin)
    : "${LINKEDIN_ACCESS_TOKEN:=}" "${LINKEDIN_AUTHOR_URN:=}"
    if [ -z "$LINKEDIN_ACCESS_TOKEN" ] || [ -z "$LINKEDIN_AUTHOR_URN" ]; then
      echo "LINKEDIN_ACCESS_TOKEN ou LINKEDIN_AUTHOR_URN absent — repli manuel" >&2
      exit 3
    fi
    # L'API Posts impose d'échapper ces caractères réservés dans 'commentary'.
    # '#' est laissé intact pour que les hashtags restent cliquables.
    ESCAPED="$(printf '%s' "$BODY" | sed 's/[\\|{}@[\]()<>*_~]/\\&/g')"
    PAYLOAD="$(jq -n \
      --arg author "$LINKEDIN_AUTHOR_URN" \
      --arg text "$ESCAPED" \
      '{author:$author, commentary:$text, visibility:"PUBLIC",
        distribution:{feedDistribution:"MAIN_FEED", targetEntities:[], thirdPartyDistributionChannels:[]},
        lifecycleState:"PUBLISHED", isReshareDisabledByAuthor:false}')"

    RESP_HEADERS="$(mktemp)"
    HTTP="$(curl -sS -o /tmp/li-body.$$ -D "$RESP_HEADERS" -w '%{http_code}' \
      -X POST 'https://api.linkedin.com/rest/posts' \
      -H "Authorization: Bearer $LINKEDIN_ACCESS_TOKEN" \
      -H "LinkedIn-Version: ${LINKEDIN_API_VERSION:-202608}" \
      -H 'X-Restli-Protocol-Version: 2.0.0' \
      -H 'Content-Type: application/json' \
      --data-binary "$PAYLOAD")"

    if [ "$HTTP" != "201" ] && [ "$HTTP" != "200" ]; then
      echo "LinkedIn a refusé (HTTP $HTTP) :" >&2; cat "/tmp/li-body.$$" >&2; echo >&2
      rm -f "/tmp/li-body.$$" "$RESP_HEADERS"; exit 2
    fi
    URN="$(grep -i '^x-restli-id:' "$RESP_HEADERS" | tr -d '\r' | awk '{print $2}')"
    rm -f "/tmp/li-body.$$" "$RESP_HEADERS"
    echo "publié : https://www.linkedin.com/feed/update/${URN}"
    ;;

  x)
    : "${X_ACCESS_TOKEN:=}"
    if [ -z "$X_ACCESS_TOKEN" ]; then echo "X_ACCESS_TOKEN absent — repli manuel" >&2; exit 3; fi
    OUT="$(curl -sS -w '\n%{http_code}' -X POST 'https://api.x.com/2/tweets' \
      -H "Authorization: Bearer $X_ACCESS_TOKEN" -H 'Content-Type: application/json' \
      --data-binary "$(jq -n --arg t "$BODY" '{text:$t}')")"
    HTTP="$(printf '%s' "$OUT" | tail -n1)"
    if [ "$HTTP" != "201" ]; then echo "X a refusé (HTTP $HTTP) :" >&2; printf '%s\n' "$OUT" >&2; exit 2; fi
    echo "publié : $(printf '%s' "$OUT" | sed '$d' | jq -r '.data.id')"
    ;;

  docs)
    echo "canal 'docs' : la publication se fait par fusion sur main, pas par ce script." >&2
    exit 3
    ;;

  reddit|hn|newsletter)
    echo "canal '$CANAL' : aucun accès automatisé configuré — repli manuel" >&2
    exit 3
    ;;

  *)
    echo "canal inconnu : $CANAL" >&2; exit 1 ;;
esac
