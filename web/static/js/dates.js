(function () {
  var nodes = document.querySelectorAll("time.md-date");
  if (!nodes.length) return;

  function pad(n) {
    return String(n).padStart(2, "0");
  }

  function utcOffset(d) {
    var min = -d.getTimezoneOffset();
    var sign = min >= 0 ? "+" : "-";
    var abs = Math.abs(min);
    return "UTC" + sign + pad(Math.floor(abs / 60)) + ":" + pad(abs % 60);
  }

  function tzShort(d) {
    var parts = new Intl.DateTimeFormat(undefined, { timeZoneName: "short" }).formatToParts(d);
    var name = "";
    for (var i = 0; i < parts.length; i++) {
      if (parts[i].type === "timeZoneName") name = parts[i].value;
    }
    if (/^[A-Za-z]{2,5}$/.test(name)) return name;
    return utcOffset(d);
  }

  function countParts(from, to) {
    if (to <= from) return null;
    var cursor = new Date(from.getTime());
    var y = 0;
    var mo = 0;
    for (;;) {
      var ny = new Date(cursor.getTime());
      ny.setFullYear(ny.getFullYear() + 1);
      if (ny > to) break;
      cursor = ny;
      y++;
    }
    for (;;) {
      var nm = new Date(cursor.getTime());
      nm.setMonth(nm.getMonth() + 1);
      if (nm > to) break;
      cursor = nm;
      mo++;
    }
    var ms = to - cursor;
    var d = Math.floor(ms / 86400000);
    ms -= d * 86400000;
    var h = Math.floor(ms / 3600000);
    ms -= h * 3600000;
    var mi = Math.floor(ms / 60000);
    ms -= mi * 60000;
    var s = Math.floor(ms / 1000);
    var bits = [];
    if (y) bits.push(y + "y");
    if (mo) bits.push(mo + "m");
    bits.push(d + "d", pad(h) + "h", pad(mi) + "m", pad(s) + "s");
    return bits.join(" ");
  }

  var pretty = {
    local: { weekday: "long", year: "numeric", month: "long", day: "numeric", hour: "numeric", minute: "2-digit" },
    "24h": { year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", hourCycle: "h23" },
    "12h": { year: "numeric", month: "short", day: "numeric", hour: "numeric", minute: "2-digit", hourCycle: "h12" }
  };

  function tick() {
    var now = new Date();
    for (var i = 0; i < nodes.length; i++) {
      var el = nodes[i];
      var d = new Date(el.getAttribute("datetime"));
      if (isNaN(d.getTime())) continue;
      var kind = el.getAttribute("data-fmt");
      if (kind === "count") {
        el.textContent = countParts(now, d) || "now";
        continue;
      }
      if (pretty[kind]) {
        var text = d.toLocaleString(undefined, pretty[kind]);
        if (kind === "24h" || kind === "12h") text += " " + tzShort(d);
        el.textContent = text;
      }
    }
  }

  tick();
  for (var j = 0; j < nodes.length; j++) {
    if (nodes[j].getAttribute("data-fmt") === "count") {
      setInterval(tick, 1000);
      break;
    }
  }
})();
