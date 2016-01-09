# Package tarballs per arch.
# FIXME: Parameterize this, but not important right now

set -e

# Geese?
GOOSES="darwin linux freebsd"
GOARCHS="amd64"

GIT_COMMIT_SHA1=`git rev-parse --short HEAD`

mkdir -p artifacts

ARTIFACTS=`pwd`/artifacts

for os in ${GOOSES}
do
    for arch in ${GOARCHS}
    do
        echo "**** Building $os.$arch ****"
        BINDIR="bin/$os/$arch"
        GOOS=$os GOARCH=$arch godep go build -ldflags "-X 'main.minversion=$(echo ${CIRCLE_BUILD_NUM})'" -o bin/$os/$arch/henchman
        cp -R modules ${BINDIR} 
        cd ${BINDIR}
        tar -cvf "henchman.${GIT_COMMIT_SHA1}.${os}.${arch}.tar.gz" henchman modules
        cp *.tar.gz ${ARTIFACTS}
        cd -
    done
done

echo "Copying artifacts"
cp -r ${ARTIFACTS}/* $CIRCLE_ARTIFACTS

echo "Cleaning up..."
rm -rf bin/
