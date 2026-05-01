# Tibiantis Assistant

Tibiantis Assistant is a collection of quality of life tools for Tibiants.

It doesn't automate the gameplay or read the game memory, so it shouldn't get you banned. That said, use it at your own risk.

## Installation

You can download the latest binary from the Releases section.

Alternatively, you're free to audit and compile the code yourself:

* Install the Go SDK: https://go.dev/doc/install
* Download the source code: https://github.com/s5i/tassist/archive/refs/heads/main.zip
* Build the binary: `GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui .`

## Usage

* Startup: The application opens a webpage in your default browser.
* Quitting: The application quits automatically 15 seconds after receiving the last ping from the browser.

### Account Switcher

* "Store" stores the current saved credentials.
  * Note: Tibiantis saves the credentials when you enter the game with a character. Getting to the character list is not enough.
* "Load" restores previously stored credentials.
* Double-clicking an entry allows you to rename it.
* Technical details:
  * Tibiantis uses Windows Registry; the credentials live under `HKEY_CURRENT_USER\SOFTWARE[tab]ibiantis\Credentials`. Yes, they failed to escape the "T" letter.
  * Tibiantis Assistant stores the Base64-encoded registry entries under `%AppData%\TAssistant\accounts.yaml`.

### Experience Tracker

* You need to have the Skills tab open with the "Experience" line being visible.
* "Level" and "Experience" display the current level and experience, respectively.
* "Exp/h" displays the experience per hour since the session started.
* "Remaining" displays the experience needed to reach the next level.
* "Start"/"Stop": controls whether the experience tracking session is active.
* "Pause"/"Unpause" controls whether the experience tracking session is paused; while paused, neither experience changes nor time elapsed is counted towards exp/h.
* "Reset" resets the experience tracking session.
* Known issues:
  * If exp detection is not working despite an active session, try re-focusing the Tibiantis client window (eg. by alt-tabbing into another window and back).
* Technical details:
  * TAssist takes a screenshot of a window titled "Tibiantis" every 2 seconds, searches for "Experience" on the right panel, and then runs OCR on the line.

### Screenshots

![Local webpage view](https://raw.github.com/s5i/tassist/main/example.png)

## Issues?

* Contact: shyymzinhu @ Discord.
* UI issues: attach DevTools console output (F12 on most browsers).
* Crash: attach the contents of `%Temp%\tassist\tassist.log`.
