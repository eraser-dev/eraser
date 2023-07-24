#!/bin/bash
# borrowed from sigs.k8s.io/cluster-api-provider-azure/hack/version.sh and modified

set -o errexit
set -o pipefail

version::get_version_vars() {
    # shellcheck disable=SC1083
    GIT_COMMIT="$(git rev-parse HEAD^{commit})"

    if git_status=$(git status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
        GIT_TREE_STATE="clean"
    else
        GIT_TREE_STATE="dirty"
    fi

    # borrowed from k8s.io/hack/lib/version.sh
    # Use git describe to find the version based on tags.
    if GIT_VERSION=$(git describe --tags --abbrev=14 2>/dev/null); then
        # This translates the "git describe" to an actual semver.org
        # compatible semantic version that looks something like this:
        #   v1.1.0-alpha.0.6+84c76d1142ea4d
        # shellcheck disable=SC2001
        DASHES_IN_VERSION=$(echo "${GIT_VERSION}" | sed "s/[^-]//g")
        if [[ "${DASHES_IN_VERSION}" == "---" ]] ; then
            # We have distance to subversion (v1.1.0-subversion-1-gCommitHash)
            # shellcheck disable=SC2001
            GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-\([0-9]\{1,\}\)-g\([0-9a-f]\{14\}\)$/.\1\-\2/")
        elif [[ "${DASHES_IN_VERSION}" == "--" ]] ; then
            # We have distance to base tag (v1.1.0-1-gCommitHash)
            # shellcheck disable=SC2001
            GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-g\([0-9a-f]\{14\}\)$/-\1/")
            # TODO: What should the output of this command look like?
            # For example, v1.1.0-32-gfeb4736460af8f maps to v1.1.0-32-f, do we want the trailing "-f" or not?
        fi

        if [[ "${GIT_TREE_STATE}" == "dirty" ]]; then
            # git describe --dirty only considers changes to existing files, but
            # that is problematic since new untracked .go files affect the build,
            # so use our idea of "dirty" from git status instead.
            GIT_VERSION+="-dirty"
        fi

        # Try to match the "git describe" output to a regex to try to extract
        # the "major" and "minor" versions and whether this is the exact tagged
        # version or whether the tree is between two tagged versions.
        if [[ "${GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?([-].*)?([+].*)?$ ]]; then
            GIT_MAJOR=${BASH_REMATCH[1]}
            GIT_MINOR=${BASH_REMATCH[2]}
        fi

        # If GIT_VERSION is not a valid Semantic Version, then exit with error
        if ! [[ "${GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
            echo "GIT_VERSION should be a valid Semantic Version. Current value: ${GIT_VERSION}"
            echo "Please see more details here: https://semver.org"
            exit 1
        fi
    fi

    if [[ -z ${SOURCE_DATE_EPOCH} ]]; then
        SOURCE_DATE="$(git show -s --format=%cI HEAD)"
        SOURCE_DATE_EPOCH="$(date -u --date "${SOURCE_DATE}" +%s)"
    fi
}

# Prints the value that needs to be passed to the -ldflags parameter of go build
version::ldflags() {
    version::get_version_vars

    local -a ldflags
    function add_ldflag() {
        local key=${1}
        local val=${2}
        ldflags+=(
            "-X 'github.com/eraser-dev/eraser/version.${key}=${val}'"
        )
    }

    add_ldflag "buildTime" "${SOURCE_DATE_EPOCH}"
    add_ldflag "vcsCommit" "${GIT_COMMIT}"
    add_ldflag "vcsState" "${GIT_TREE_STATE}"

    if [[ ! -z ${GIT_VERSION} ]]; then
        add_ldflag "BuildVersion" "${GIT_VERSION}"
        add_ldflag "vcsMajor" "${GIT_MAJOR}"
        add_ldflag "vcsMinor" "${GIT_MINOR}"
    elif [[ -n $1 ]]; then
        add_ldflag "BuildVersion" "$1"
    fi

    # The -ldflags parameter takes a single string, so join the output.
    echo "${ldflags[*]-}"
}

version::ldflags $1
