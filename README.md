# TAssistant

TAssistant is a collection of quality of life tools for some OT servers (Tibiantis, Relic).

It doesn't automate the gameplay or read the game memory, so it shouldn't get you banned. That said, use it at your own risk.

## Installation

You can download the latest binary from the Releases section.

Alternatively, you're free to audit and compile the code yourself:

* Install the Go SDK: https://go.dev/doc/install
* Download the source code: https://github.com/s5i/tassist/archive/refs/heads/main.zip
* Build the binary: `GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui .`

## Usage

* Startup: The application opens a webpage in your default browser.
* Quitting: The application quits automatically ~15-20 seconds after closing the browser tab.

### Account Switcher

* "Store" stores the current saved credentials.
  * Tibiantis saves the credentials when you enter the game with a character. Getting to the character list is not enough.
* "Load" restores previously stored credentials.
* Double-clicking an entry allows you to rename it.
* Technical details:
  * TAssistant stores the Base64-encoded registry entries under `%AppData%\TAssistant\accounts.yaml`.
  * Tibiantis uses Windows Registry; the credentials live under `HKEY_CURRENT_USER\SOFTWARE[tab]ibiantis\Credentials`. Yes, they failed to escape the "T" letter.
  * Relic uses Windows Registry; the credentials live under `HKEY_CURRENT_USER\SOFTWARE\Tibia Relic\Credentials`.

### Experience Tracker

* You need to have the Skills tab open with the "Experience" line being visible.
* "Level" and "Experience" display the current level and experience, respectively.
* "Remaining" displays the experience needed to reach the next level.
* "Session exp/h", "Session exp", "Session duration" display the experience per hour, gained experience, and time elapsed since the session started, respectively.
* "Start"/"Stop": controls whether the session is active.
* "Pause"/"Unpause" controls whether the session is paused; while paused, neither experience changes nor time elapsed is counted towards exp/h.
* "Reset" resets the session.
* Known issues:
  * If exp detection is not working despite an active session, try re-focusing the game window (eg. by alt-tabbing into another window and back).
* Technical details:
  * TAssistant takes a screenshot of a window matching the OT-specific window title every 2 seconds, searches for "Experience" on the right panel, and then runs OCR on the line.

### Ping

* "RTT" displays your round-trip-time (a.k.a. ping) to Ancestra.
* "Packet loss" displays the percentage of ping packets lost within the last 20 seconds.
* Secondary values in parentheses, if present, indicate the statistics measured against proxies.
* Technical details:
  * Ancestra IP is assumed to be `51.89.155.163`.
  * Concordia IP is assumed to be `57.129.145.195`.
  * Relic IP is assumed to be `104.156.244.186`, with extra proxies at `216.238.121.95`, `45.32.218.87`, `95.179.154.226`.

### Screenshots

![Local webpage view](https://raw.github.com/s5i/tassist/main/example.png)

## Issues?

* Contact: shyymzinhu @ Discord.
* UI issues: attach DevTools console output (F12 on most browsers).
* Crash / other weird issues: attach the contents of `%Temp%\tassist\tassist.log`.
