PROJECT=$1

echo "Launching build machine."
DIR="$(dirname "$0")"
RAND="$(openssl rand -hex 5)"
ZONES=("us-central1-b" "us-central1-c" "europe-west1-d" "us-east1-d")

for zone in "${ZONES[@]}"; do
gcloud compute instances create "v2raygeoip-${RAND}" \
    --machine-type=n1-standard-1 \
    --metadata-from-file=startup-script=${DIR}/generate.sh \
    --zone=${zone} \
    --project ${PROJECT}
if [ $? -eq 0 ]; then
  exit 0
fi
done
