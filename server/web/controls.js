// HornetPad control map. Every command is a DCS-BIOS control from the
// F/A-18C module (Scripts/DCS-BIOS/lib/modules/aircraft_modules/FA-18C_hornet.lua).
//
// Item types:
//   set    - one tap sends `value` (rotary selector positions)
//   rocker - press sends `value`, release sends `center` (spring-loaded switches)
//   pot    - knob: sends `step` ("+3200" / "-3200") repeatedly while held
//
// OSB numbering (real Hornet): PB1 = left side bottom, then clockwise.
//   left  1-5  bottom -> top
//   top   6-10 left -> right
//   right 11-15 top -> bottom
//   bottom 16-20 right -> left   (PB18 = MENU, bottom center)

const POT_STEP = 3200;

window.HORNETPAD_UNITS = {
  LDDI: {
    title: "LEFT DDI",
    osbPrefix: "LEFT_DDI_PB_",
    groups: [
      { name: "BRT SEL", items: [
        { type: "set", label: "OFF", cmd: "LEFT_DDI_BRT_SELECT", value: "0" },
        { type: "set", label: "NGT", cmd: "LEFT_DDI_BRT_SELECT", value: "1" },
        { type: "set", label: "DAY", cmd: "LEFT_DDI_BRT_SELECT", value: "2" },
      ]},
      { name: "BRT", items: [
        { type: "pot", label: "−", cmd: "LEFT_DDI_BRT_CTL", step: -POT_STEP },
        { type: "pot", label: "+", cmd: "LEFT_DDI_BRT_CTL", step: POT_STEP },
      ]},
      { name: "CONT", items: [
        { type: "pot", label: "−", cmd: "LEFT_DDI_CONT_CTL", step: -POT_STEP },
        { type: "pot", label: "+", cmd: "LEFT_DDI_CONT_CTL", step: POT_STEP },
      ]},
      { name: "HDG", items: [
        { type: "rocker", label: "◀", cmd: "LEFT_DDI_HDG_SW", value: "0", center: "1" },
        { type: "rocker", label: "▶", cmd: "LEFT_DDI_HDG_SW", value: "2", center: "1" },
      ]},
      { name: "CRS", items: [
        { type: "rocker", label: "◀", cmd: "LEFT_DDI_CRS_SW", value: "0", center: "1" },
        { type: "rocker", label: "▶", cmd: "LEFT_DDI_CRS_SW", value: "2", center: "1" },
      ]},
    ],
  },

  AMPCD: {
    title: "AMPCD",
    osbPrefix: "AMPCD_PB_",
    groups: [
      { name: "BRT", items: [
        { type: "pot", label: "−", cmd: "AMPCD_BRT_CTL", step: -POT_STEP },
        { type: "pot", label: "+", cmd: "AMPCD_BRT_CTL", step: POT_STEP },
      ]},
      { name: "N / D", items: [
        { type: "rocker", label: "NGT", cmd: "AMPCD_NIGHT_DAY", value: "0", center: "1" },
        { type: "rocker", label: "DAY", cmd: "AMPCD_NIGHT_DAY", value: "2", center: "1" },
      ]},
      { name: "SYM", items: [
        { type: "rocker", label: "▼", cmd: "AMPCD_SYM_SW", value: "0", center: "1" },
        { type: "rocker", label: "▲", cmd: "AMPCD_SYM_SW", value: "2", center: "1" },
      ]},
      { name: "CONT", items: [
        { type: "rocker", label: "▼", cmd: "AMPCD_CONT_SW", value: "0", center: "1" },
        { type: "rocker", label: "▲", cmd: "AMPCD_CONT_SW", value: "2", center: "1" },
      ]},
      { name: "GAIN", items: [
        { type: "rocker", label: "▼", cmd: "AMPCD_GAIN_SW", value: "0", center: "1" },
        { type: "rocker", label: "▲", cmd: "AMPCD_GAIN_SW", value: "2", center: "1" },
      ]},
    ],
  },

  RDDI: {
    title: "RIGHT DDI",
    osbPrefix: "RIGHT_DDI_PB_",
    groups: [
      { name: "BRT SEL", items: [
        { type: "set", label: "OFF", cmd: "RIGHT_DDI_BRT_SELECT", value: "0" },
        { type: "set", label: "NGT", cmd: "RIGHT_DDI_BRT_SELECT", value: "1" },
        { type: "set", label: "DAY", cmd: "RIGHT_DDI_BRT_SELECT", value: "2" },
      ]},
      { name: "BRT", items: [
        { type: "pot", label: "−", cmd: "RIGHT_DDI_BRT_CTL", step: -POT_STEP },
        { type: "pot", label: "+", cmd: "RIGHT_DDI_BRT_CTL", step: POT_STEP },
      ]},
      { name: "CONT", items: [
        { type: "pot", label: "−", cmd: "RIGHT_DDI_CONT_CTL", step: -POT_STEP },
        { type: "pot", label: "+", cmd: "RIGHT_DDI_CONT_CTL", step: POT_STEP },
      ]},
    ],
  },
};

// Returns {side, slot} for OSB n (1-20); slot counts left->right / top->bottom.
window.hornetpadOsbPosition = function (n) {
  if (n <= 5) return { side: "left", slot: 5 - n };
  if (n <= 10) return { side: "top", slot: n - 6 };
  if (n <= 15) return { side: "right", slot: n - 11 };
  return { side: "bottom", slot: 20 - n };
};
