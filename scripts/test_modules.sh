#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

if [ -f .env ]; then
  echo "[test-modules] Loading .env"
  # shellcheck disable=SC1091
  set -a
  source .env
  set +a
fi

if [ -n "${GO_TEST_FLAGS:-}" ]; then
  # shellcheck disable=SC2206
  GO_TEST_FLAGS_ARRAY=(${GO_TEST_FLAGS})
else
  GO_TEST_FLAGS_ARRAY=()
fi

MODULE_MATRIX=(
  "decision:./decision:Real"
  "api:./api:Real"
  "trader:./trader:Real"
  "backtest:./backtest:Real"
  "config:./config:Real"
  "manager:./manager:Real"
)

REQUESTED_MODULE="${MODULE:-}"
EXIT_CODE=0

for entry in "${MODULE_MATRIX[@]}"; do
  IFS=":" read -r module_name module_path run_pattern <<<"${entry}"

  if [ -n "${REQUESTED_MODULE}" ] && [ "${REQUESTED_MODULE}" != "${module_name}" ]; then
    continue
  fi

  echo "[test-modules] >>> ${module_name}: go test ${module_path} -run ${run_pattern}"
  if ! go test "${GO_TEST_FLAGS_ARRAY[@]}" "${module_path}" -run "${run_pattern}"; then
    echo "[test-modules] !!! ${module_name} failed"
    EXIT_CODE=1
  else
    echo "[test-modules] <<< ${module_name} passed"
  fi
  echo
done

if [ -n "${REQUESTED_MODULE}" ]; then
  FOUND=0
  for entry in "${MODULE_MATRIX[@]}"; do
    IFS=":" read -r module_name _ <<<"${entry}"
    if [ "${module_name}" = "${REQUESTED_MODULE}" ]; then
      FOUND=1
      break
    fi
  done
  if [ ${FOUND} -eq 0 ]; then
    echo "[test-modules] Unknown module '${REQUESTED_MODULE}'. Known modules: ${MODULE_MATRIX[*]}" >&2
    exit 2
  fi
fi

exit ${EXIT_CODE}
