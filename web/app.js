const LINE_COLORS = {
  1: "#ee352e", 2: "#ee352e", 3: "#ee352e",
  4: "#00933c", 5: "#00933c", 6: "#00933c",
  7: "#b933ad",
  A: "#0039a6", C: "#0039a6", E: "#0039a6",
  B: "#ff6319", D: "#ff6319", F: "#ff6319", M: "#ff6319",
  G: "#6cbe45",
  J: "#996633", Z: "#996633",
  L: "#a7a9ac",
  N: "#fccc0a", Q: "#fccc0a", R: "#fccc0a", W: "#fccc0a",
  S: "#808183", FS: "#808183", GS: "#808183", H: "#808183",
  SI: "#0039a6", SIR: "#0039a6",
};

function transitApp() {
  return {
    currentMode: "subway",
    radius: "800",
    zipValue: "",
    lastZip: null,
    lastCoords: null,
    loading: false,
    locating: false,
    error: null,
    locationText: null,
    stations: [],
    busStops: [],
    lastUpdated: null,
    clock: "",
    _refreshTimer: null,
    favorites: JSON.parse(localStorage.getItem("emteeayy_favorites") || "[]"),
    alerts: [],
    alertsExpanded: false,

    init() {
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);
      this.$nextTick(() => this.$refs.zipInput.focus());
    },

    updateClock() {
      this.clock = new Date().toLocaleTimeString("en-US", {
        hour: "numeric",
        minute: "2-digit",
        second: "2-digit",
        hour12: true,
      });
    },

    setMode(mode) {
      this.currentMode = mode;
      this.alerts = [];
      this.alertsExpanded = false;
      if (mode === "saved") {
        this.loadFavorites();
      } else if (this.lastZip) {
        this.fetchByZip(this.lastZip);
      } else if (this.lastCoords) {
        this.fetchByCoords(this.lastCoords.lat, this.lastCoords.lng);
      }
    },

    submitZip() {
      const zip = this.zipValue.trim();
      if (!/^\d{5}$/.test(zip)) {
        this.error = "Please enter a valid 5-digit zip code";
        return;
      }
      this.fetchByZip(zip);
    },

    useLocation() {
      if (!navigator.geolocation) {
        this.error = "Geolocation is not supported by your browser";
        return;
      }
      this.locating = true;
      this.error = null;
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          this.locating = false;
          this.fetchByCoords(pos.coords.latitude, pos.coords.longitude);
        },
        (err) => {
          this.locating = false;
          this.error =
            err.code === err.PERMISSION_DENIED
              ? "Location access denied. Please allow location access or enter a zip code."
              : "Could not get your location";
        },
        { enableHighAccuracy: true, timeout: 10000 }
      );
    },

    async fetchByZip(zip) {
      this.loading = true;
      this.error = null;
      this.locationText = null;
      this.lastZip = zip;
      this.lastCoords = null;

      const endpoint =
        this.currentMode === "subway"
          ? `/transit/subway/near/${zip}?radius=${this.radius}`
          : `/transit/bus/near/${zip}?radius=${this.radius}`;

      try {
        const resp = await fetch(endpoint);
        const data = await resp.json();
        if (!resp.ok)
          throw new Error(data.message || data.error || "Failed to fetch");
        if (data.location) {
          const city = data.location.city || "";
          const borough = data.location.borough || "NYC";
          const display =
            city && city !== borough ? `${city}, ${borough}` : borough;
          this.locationText = `${display} (${zip})`;
        }
        this.applyData(data);
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },

    async fetchByCoords(lat, lng) {
      this.loading = true;
      this.error = null;
      this.lastCoords = { lat, lng };
      this.lastZip = null;

      const endpoint =
        this.currentMode === "subway"
          ? `/transit/subway/near?lat=${lat}&lng=${lng}&radius=${this.radius}`
          : `/transit/bus/near?lat=${lat}&lng=${lng}&radius=${this.radius}`;

      try {
        const resp = await fetch(endpoint);
        const data = await resp.json();
        if (!resp.ok)
          throw new Error(data.message || data.error || "Failed to fetch");
        this.locationText = `Near ${lat.toFixed(4)}, ${lng.toFixed(4)}`;
        this.applyData(data);
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },

    applyData(data) {
      if (this.currentMode === "subway" || this.currentMode === "saved") {
        this.stations = data.stations || [];
        this.busStops = [];
        const routes = new Set();
        for (const s of this.stations) {
          for (const a of s.northbound || []) routes.add(a.route);
          for (const a of s.southbound || []) routes.add(a.route);
        }
        if (routes.size > 0) this.fetchAlerts([...routes]);
        else this.alerts = [];
      } else {
        this.stations = [];
        this.alerts = [];
        const byStop = {};
        for (const arr of data.arrivals || []) {
          const key = arr.stop_name || arr.stop_id;
          if (!byStop[key]) byStop[key] = { name: key, arrivals: [] };
          byStop[key].arrivals.push(arr);
        }
        this.busStops = Object.values(byStop);
      }
      this.lastUpdated = new Date().toLocaleTimeString("en-US", {
        hour: "numeric",
        minute: "2-digit",
        second: "2-digit",
        hour12: true,
      });
      this.scheduleRefresh();
    },

    refresh() {
      if (this.currentMode === "saved") this.loadFavorites();
      else if (this.lastZip) this.fetchByZip(this.lastZip);
      else if (this.lastCoords)
        this.fetchByCoords(this.lastCoords.lat, this.lastCoords.lng);
    },

    scheduleRefresh() {
      clearInterval(this._refreshTimer);
      this._refreshTimer = setInterval(() => this.refresh(), 60000);
    },

    toggleFavorite(stopId, stopName) {
      const idx = this.favorites.findIndex((f) => f.id === stopId);
      if (idx >= 0) {
        this.favorites.splice(idx, 1);
      } else {
        this.favorites.push({ id: stopId, name: stopName });
      }
      localStorage.setItem(
        "emteeayy_favorites",
        JSON.stringify(this.favorites)
      );
      if (this.favorites.length === 0 && this.currentMode === "saved") {
        this.setMode("subway");
      }
    },

    isFavorite(stopId) {
      return this.favorites.some((f) => f.id === stopId);
    },

    async loadFavorites() {
      if (this.favorites.length === 0) {
        this.stations = [];
        this.lastUpdated = null;
        return;
      }
      this.loading = true;
      this.error = null;
      this.locationText = "Saved stations";
      const ids = this.favorites.map((f) => f.id).join(",");
      try {
        const resp = await fetch(`/transit/subway/arrivals?stops=${ids}`);
        const data = await resp.json();
        if (!resp.ok)
          throw new Error(data.message || data.error || "Failed to fetch");
        this.applyData(data);
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },

    async fetchAlerts(routes) {
      try {
        const resp = await fetch(
          `/transit/subway/alerts?routes=${routes.join(",")}`
        );
        const data = await resp.json();
        if (resp.ok && data.alerts) {
          this.alerts = data.alerts;
        }
      } catch {
        // Alerts are non-critical
      }
    },

    getLineColor(line) {
      return LINE_COLORS[line] || "#888";
    },

    isLightLine(line) {
      return ["N", "Q", "R", "W"].includes(line);
    },

    formatTime(minutes) {
      if (minutes === 0) return "now";
      if (minutes === 1) return "1 min";
      return `${minutes} min`;
    },
  };
}
