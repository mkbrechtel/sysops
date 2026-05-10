#!/bin/sh
# SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
#
# Container entrypoint: regenerate the sidebar (the markdown content is
# bind-mounted from the host clone, so it may have changed since the
# image was built), then exec the supplied command.

set -eu

/usr/local/share/devops-website/scripts/build-sidebar.sh

exec "$@"
