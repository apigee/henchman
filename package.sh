# Package tarballs per arch.
# FIXME: Parameterize this, but not important right now

set -e

# Geese?
GOOSES="darwin linux freebsd"
GOARCHS="amd64"

GIT_COMMIT_SHA1=`git rev-parse --short HEAD`

mkdir -p artefacts

ARTEFACTS=`pwd`/artefacts

for os in ${GOOSES}
do
    for arch in ${GOARCHS}
    do
        echo "**** Building $os.$arch ****"
        BINDIR="bin/$os/$arch"
        GOOS=$os GOARCH=$arch godep go build -o bin/$os/$arch/henchman
        cp -R modules ${BINDIR} 
        cd ${BINDIR}
        tar -cvf "henchman.${GIT_COMMIT_SHA1}.${os}.${arch}.tar.gz" henchman modules
        cp *.tar.gz ${ARTEFACTS}
        cd -
    done
done

echo "Cleaning up..."
rm -rf bin/
