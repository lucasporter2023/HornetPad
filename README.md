# HornetPad

A Falcon BMS Control–style touch panel for the DCS F/A-18C Hornet. Your Android tablet shows the
**Left DDI, AMPCD and Right DDI** side by side in landscape, with all 20 pushbuttons around each
display plus the brightness/contrast knobs, HDG/CRS switches and AMPCD rockers. The tablet connects
to the PC over a **USB cable**.

```
 DCS ──exports──> virtual display ──screen capture──> HornetPad.exe ──USB (adb)──> tablet app
 DCS <──────────── DCS-BIOS (UDP 7778) <──────────── button presses <────────────── tablet app
```

| Part | What it is |
|---|---|
| `HornetPad.exe` | Windows server. Captures the three exports, streams them, sends DCS-BIOS commands, sets up the USB link. |
| `HornetPad.apk` | Android app. Full screen, always on, haptic buttons, reconnects automatically. |
| `dcs/HornetPad.lua` | DCS monitor setup that puts the exports where HornetPad captures them. |

Nothing here touches DCS core files, so it will not fail the DCS Integrity Check.

---

## 1. One-time PC setup

1. **DCS-BIOS** (DCS-Skunkworks version) must be installed. If you already use it for other
   panels, you're done.
2. **Monitor setup.** Copy `dcs/HornetPad.lua` to `Saved Games\DCS\Config\MonitorSetup\`.
   In DCS go to *Options > System*, set *Monitors* to **HornetPad** and *Resolution* to
   **3840x1080**.
   - It uses the same layout as your MATRIC setup: 1920x1080 primary, 1920x1080 virtual
     display to the right, 620x620 exports with 20 px gaps. If you'd rather keep your
     existing monitor lua, just set the `displays` coordinates in `config.json` to match it.
3. **adb (USB link).** Download *SDK Platform-Tools for Windows* from
   <https://developer.android.com/tools/releases/platform-tools>, and copy `adb.exe`,
   `AdbWinApi.dll` and `AdbWinUsbApi.dll` into the same folder as `HornetPad.exe`.
4. Run `HornetPad.exe` once. It creates `config.json` next to itself.
   To check the capture regions, run `HornetPad.exe -snapshot` while sitting in a Hornet
   cockpit: it saves `snapshot_LDDI.png`, `snapshot_AMPCD.png` and `snapshot_RDDI.png`.
   Each should show exactly one display, with no black edges.

## 2. One-time tablet setup

1. Turn on **Developer options**: *Settings > About tablet*, tap *Build number* 7 times.
2. *Settings > System > Developer options*: turn on **USB debugging**.
3. Plug the tablet into the PC with HornetPad.exe running, and tap **Allow** (tick
   *Always allow from this computer*) on the "Allow USB debugging?" prompt.
4. Install the app: `adb install HornetPad.apk` (from the HornetPad folder), or copy the APK to
   the tablet and open it.

**No APK yet?** Open Chrome on the tablet and go to `http://127.0.0.1:8765` while it's plugged
in. It is the exact same panel. Use *Add to Home screen* to get a full-screen icon.

## 3. Every flight

1. Start `HornetPad.exe` (before or after DCS, it doesn't matter).
2. Plug in the tablet. HornetPad sets up the USB link and opens the app on the tablet by itself.
3. Fly. The console window shows the USB state, frame rate and the last command sent.

---

## Controls

**OSBs** use the real Hornet numbering: PB1 is the bottom button on the left side, counting
clockwise (6–10 across the top, 11–15 down the right, 16–20 right to left along the bottom,
so PB18 is MENU). The small numbers on the buttons are just for reference. Add `?nonumbers` to the
URL to hide them.

| Strip | Controls | DCS-BIOS |
|---|---|---|
| Left DDI | BRT SEL OFF/NGT/DAY, BRT −/+, CONT −/+, HDG ◀/▶, CRS ◀/▶ | `LEFT_DDI_BRT_SELECT`, `LEFT_DDI_BRT_CTL`, `LEFT_DDI_CONT_CTL`, `LEFT_DDI_HDG_SW`, `LEFT_DDI_CRS_SW` |
| AMPCD | BRT −/+, NGT/DAY, SYM ▼/▲, CONT ▼/▲, GAIN ▼/▲ | `AMPCD_BRT_CTL`, `AMPCD_NIGHT_DAY`, `AMPCD_SYM_SW`, `AMPCD_CONT_SW`, `AMPCD_GAIN_SW` |
| Right DDI | BRT SEL OFF/NGT/DAY, BRT −/+, CONT −/+ | `RIGHT_DDI_BRT_SELECT`, `RIGHT_DDI_BRT_CTL`, `RIGHT_DDI_CONT_CTL` |

Knobs keep turning while held. HDG/CRS and the AMPCD rockers are held while your finger is down and
spring back to center on release. Every control is defined in `server/web/controls.js` if you want
to change or add anything (rebuild the exe afterwards).

## config.json

| Key | Default | Meaning |
|---|---|---|
| `port` | 8765 | Port used over USB. (The Android app expects 8765.) |
| `allowLan` | false | `true` also lets a tablet connect over Wi-Fi at `http://<PC-IP>:8765` (USB is faster). |
| `fps` | 30 | Capture rate. 30 is smooth; 45–60 costs more CPU. |
| `jpegQuality` | 80 | Lower = less data, slightly softer image. |
| `dcsBios` | 127.0.0.1:7778 | Where DCS-BIOS listens for commands. |
| `adbPath` | "" | Full path to adb.exe if it isn't next to HornetPad.exe. |
| `autoAdb` / `autoLaunchApp` | true | Set up the USB link / open the app when the tablet is plugged in. |
| `displays` | see lua | Capture rectangles. Must match the monitor setup lua. |
| `order` | LDDI, AMPCD, RDDI | Left-to-right order on the tablet. |

## Troubleshooting

| Symptom | Fix |
|---|---|
| Console says *adb.exe not found* | Copy the 3 platform-tools files next to HornetPad.exe, or set `adbPath`. |
| *tablet found but not authorized* | Unlock the tablet and tap **Allow** on the USB debugging prompt. |
| Tablet says *Waiting for HornetPad.exe* | Check that HornetPad.exe is running and the cable is a data cable. Try a different USB port. |
| Displays show *NO SIGNAL* | You're not in a Hornet cockpit yet, or the monitor setup isn't selected in DCS. |
| Display shows the wrong area / black edges | Run `HornetPad.exe -snapshot` and adjust `displays` in config.json. |
| Buttons do nothing | DCS-BIOS isn't installed or running. Check `Saved Games\DCS\Scripts\Export.lua` has the DCS-BIOS line. |
| Picture lags | Lower `jpegQuality` to 65–70, use a USB 3 port, and keep `fps` at 30. |

## Building it yourself

- **Windows server:** install Go, then `cd server` and
  `set GOOS=windows&& go build -ldflags="-s -w" -o HornetPad.exe .`
- **Android app, option A (GitHub, no installs):** push this folder to a GitHub repository, open
  the *Actions* tab, run **Build HornetPad**, and download the *HornetPad* artifact
  (contains both `HornetPad.apk` and `HornetPad.exe`).
- **Android app, option B:** open the `android` folder in Android Studio and choose
  *Build > Build App Bundle(s) / APK(s) > Build APK(s)*.
