set -e

docker build -t pm2-test -f test/Dockerfile .

JOBS=2
OPTS="--jobs $JOBS --joblog joblog-X docker run -v `pwd`:/var/pm2 pm2-test"

ls test/e2e/cli/* | parallel $OPTS bash
