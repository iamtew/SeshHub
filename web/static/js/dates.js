(function () {
  var nodes = document.querySelectorAll("time.md-date");
  if (!nodes.length) return;

  function pad(n) {
    return String(n).padStart(2, "0");
  }

  function isoOffsetMin(iso) {
    if (!iso) return null;
    iso = String(iso).trim();
    if (/Z$/i.test(iso)) return 0;
    var m = iso.match(/([+-])(\d{2}):(\d{2})$/);
    if (!m) return null;
    return (m[1] === "-" ? -1 : 1) * (parseInt(m[2], 10) * 60 + parseInt(m[3], 10));
  }

  function utcLabel(min) {
    var sign = min >= 0 ? "+" : "-";
    var abs = Math.abs(min);
    return "UTC" + sign + pad(Math.floor(abs / 60)) + ":" + pad(abs % 60);
  }

  function zoneOffsetMin(d, tz) {
    var p = {};
    new Intl.DateTimeFormat("en-US", {
      timeZone: tz,
      hourCycle: "h23",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit"
    }).formatToParts(d).forEach(function (x) {
      p[x.type] = x.value;
    });
    return (Date.UTC(+p.year, +p.month - 1, +p.day, +p.hour, +p.minute, +p.second) - d.getTime()) / 60000;
  }

  function zoneShort(d, tz) {
    var parts = new Intl.DateTimeFormat("en-GB", {
      timeZone: tz,
      timeZoneName: "short",
      hour: "numeric"
    }).formatToParts(d);
    for (var i = 0; i < parts.length; i++) {
      if (parts[i].type === "timeZoneName" && /^[A-Za-z]{2,5}$/.test(parts[i].value)) {
        return parts[i].value;
      }
    }
    return "";
  }

  var prefer = ["Europe/Amsterdam", "Europe/Berlin", "Europe/Paris", "Europe/London", "Europe/Lisbon", "Europe/Helsinki"];
  var zoneList;
  function allZones() {
    if (zoneList) return zoneList;
    zoneList = prefer.slice();
    if (typeof Intl.supportedValuesOf === "function") {
      var all = Intl.supportedValuesOf("timeZone");
      for (var i = 0; i < all.length; i++) {
        if (zoneList.indexOf(all[i]) < 0) zoneList.push(all[i]);
      }
    }
    return zoneList;
  }

  var foundCache = {};
  function zoneFor(d, iso) {
    var want = isoOffsetMin(iso);
    if (want == null) return { tz: undefined, name: "" };
    var key = want + "@" + d.getTime();
    if (foundCache[key]) return foundCache[key];
    var list = allZones();
    var fallback = null;
    for (var i = 0; i < list.length; i++) {
      if (zoneOffsetMin(d, list[i]) !== want) continue;
      var n = zoneShort(d, list[i]);
      if (n) {
        foundCache[key] = { tz: list[i], name: n };
        return foundCache[key];
      }
      if (!fallback) fallback = { tz: list[i], name: utcLabel(want) };
    }
    foundCache[key] = fallback || { tz: undefined, name: utcLabel(want) };
    return foundCache[key];
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
    var day = Math.floor(ms / 86400000);
    ms -= day * 86400000;
    var h = Math.floor(ms / 3600000);
    ms -= h * 3600000;
    var mi = Math.floor(ms / 60000);
    ms -= mi * 60000;
    var s = Math.floor(ms / 1000);
    var bits = [];
    if (y) bits.push(y + "y");
    if (mo) bits.push(mo + "m");
    bits.push(day + "d", pad(h) + "h", pad(mi) + "m", pad(s) + "s");
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
      var iso = el.getAttribute("datetime");
      var d = new Date(iso);
      if (isNaN(d.getTime())) continue;
      var kind = el.getAttribute("data-fmt");
      if (kind === "count") {
        el.textContent = countParts(now, d) || "now";
        continue;
      }
      if (!pretty[kind]) continue;
      var opts = pretty[kind];
      if (kind === "24h" || kind === "12h") {
        var z = zoneFor(d, iso);
        if (z.tz) opts = Object.assign({ timeZone: z.tz }, pretty[kind]);
        el.textContent = d.toLocaleString("en-GB", opts) + " " + z.name;
        continue;
      }
      el.textContent = d.toLocaleString(undefined, opts);
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
