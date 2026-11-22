-- HornetPad monitor setup for DCS: F/A-18C exports on a second (virtual) display.
-- Copy to:  Saved Games\DCS\Config\MonitorSetup\HornetPad.lua
-- In DCS:   Options > System > Monitors = "HornetPad", Resolution = 3840x1080
--
-- Layout: 1920x1080 primary + 1920x1080 display directly to the right (top aligned).
-- The three exports sit on the right display as 620x620 squares with 20 px gaps.
-- If you change anything here, change "displays" in HornetPad's config.json to match.

_ = function(p) return p end
name = _('HornetPad (F/A-18C DDIs + AMPCD)')
Description = 'Primary 1920x1080 + right 1920x1080: LDDI / AMPCD / RDDI 620x620 exports'

Viewports =
{
    Center =
    {
        x = 0,
        y = 0,
        width = 1920,
        height = 1080,
        viewDx = 0,
        viewDy = 0,
        aspect = 1920 / 1080,
    }
}

-- Left DDI
LEFT_MFCD   = { x = 1920, y = 0, width = 620, height = 620 }
-- AMPCD
CENTER_MFCD = { x = 2560, y = 0, width = 620, height = 620 }
-- Right DDI
RIGHT_MFCD  = { x = 3200, y = 0, width = 620, height = 620 }

UIMainView = Viewports.Center
GU_MAIN_VIEWPORT = Viewports.Center
