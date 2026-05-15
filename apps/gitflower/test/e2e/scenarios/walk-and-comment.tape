# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
#
# Walk through the test repo's feature branch, leave a comment, set a
# verdict, save, quit. Produces both a GIF (for humans) and a cast (for
# replay).

Output walk-and-comment.gif
Output walk-and-comment.cast

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Padding 12
Set PlaybackSpeed 0.6

# Setup runs invisibly: rebuild repo + launch gitflower from the worktree.
Hide
Type "bash $GITFLOWER_E2E/setup.sh /tmp/gitflower-e2e-repo >/dev/null"
Enter
Sleep 200ms
Type "cd /tmp/gitflower-e2e-repo"
Enter
Type "git config user.email reviewer@example.com"
Enter
Type "$GITFLOWER_BIN review --base main --read-delay 200ms feature"
Enter
Sleep 800ms
Show

# Drill into Changes from the sidebar.
Space
Sleep 1s

# Walk to the next unread hunk (greet.go change).
Space
Sleep 1s

# Add a comment.
Type "c"
Sleep 400ms
Type "Nice default."
Alt+Enter
Sleep 800ms

# Approve.
Type ">"
Sleep 600ms

# Save explicitly + quit.
Type "s"
Sleep 400ms
Type "q"
Sleep 400ms
