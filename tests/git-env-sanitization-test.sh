#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"

env \
  GIT_DIR=/poison \
  git_work_tree=/lowercase-poison \
  Git_Index_File=/mixed-case-poison \
  GIT_WORK_TREE=/poison \
  GIT_CONFIG_COUNT=1 \
  GIT_CONFIG_KEY_0=core.worktree \
  GIT_CONFIG_VALUE_0=/poison \
  git_config_key_7=core.fsmonitor \
  Git_Config_Value_7=/mixed-case-poison \
  GIT_AUTHOR_NAME='Retained Author' \
  git_custom_keep='Retained Custom' \
  bash -euo pipefail -c '
    . "$1/scripts/git-env.sh"
    [[ -z ${GIT_DIR+x} ]]
    [[ -z ${git_work_tree+x} ]]
    [[ -z ${Git_Index_File+x} ]]
    [[ -z ${GIT_WORK_TREE+x} ]]
    [[ -z ${GIT_CONFIG_COUNT+x} ]]
    [[ -z ${GIT_CONFIG_KEY_0+x} ]]
    [[ -z ${GIT_CONFIG_VALUE_0+x} ]]
    [[ -z ${git_config_key_7+x} ]]
    [[ -z ${Git_Config_Value_7+x} ]]
    [[ ${GIT_AUTHOR_NAME} == "Retained Author" ]]
    [[ ${git_custom_keep} == "Retained Custom" ]]
  ' _ "$ROOT"

echo "git-env-sanitization-test: PASS"
