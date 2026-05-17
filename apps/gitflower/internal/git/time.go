// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

package git

import "time"

// nowFunc is the clock used for signature timestamps. Indirected so
// tests can pin it to a stable value when they need byte-comparable
// commit OIDs.
var nowFunc = time.Now
