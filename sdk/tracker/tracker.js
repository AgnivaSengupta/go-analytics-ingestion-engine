(function (global) {
  class AnalyticsEngine {
    constructor(config) {
      this.siteId = config.siteId;
      this.apiKey = config.apiKey;
      this.endpoint = config.endpoint || "http://localhost:8080/v1/ingest";

      this.queue = [];
      this.flushInterval = config.flushInterval || 5000;
      this.maxQueueLength = config.maxQueueLength || 10;

      this.visitorId = this._getOrSetIdentity("ae_visitor_id", localStorage);
      this.sessionId = this._getOrSetIdentity("ae_session_id", sessionStorage);

      this._startBatcher();
      this._registerUnloadHandler();
    }

    _getOrSetIdentity(key, storage) {
      let id = storage.getItem(key);
      if (!id) {
        id = "id_" + crypto.randomUUID().replace(/-/g, "");
        storage.setItem(key, id);
      }
      return id;
    }

    track(eventName, eventType, properties = {}) {
      const event = {
        event_id: "evt_" + crypto.randomUUID().replace(/-/g, ""),
        site_id: this.siteId,
        visitor_id: this.visitorId,
        session_id: this.sessionId,
        event_name: eventName,
        event_type: eventType,
        occurred_at: new Date().toISOString(),
        page_url: window.location.href,
        page_path: window.location.pathname,
        referrer: document.referrer || "",
        properties: properties,
      };

      this.queue.push(event);

      if (this.queue.length >= this.maxQueueLength) {
        this._flush();
      }
    }

    trackPageView(properties = {}) {
      this.track("page_view", "page", properties);
    }

    _startBatcher() {
      setInterval(() => {
        if (this.queue.length > 0) {
          this._flush();
        }
      }, this.flushInterval);
    }

    _registerUnloadHandler() {
      document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "hidden" && this.queue.length > 0) {
          this._flush(true);
        }
      });
    }

    async _flush(isUnloading = false) {
      if (this.queue.length === 0) return;

      const batch = [...this.queue];
      this.queue = [];

      try {
        const response = await fetch(this.endpoint, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${this.apiKey}`,
          },
          body: JSON.stringify({ events: batch }),
          keepalive: isUnloading,
        });

        if (response.status === 429 || response.status >= 500) {
          console.warn(
            "Analytics Engine: Transient error, requeuing batch. Status:",
            response.status,
          );
          this.queue = batch.concat(this.queue);
        }
        const MAX_FALLBACK_QUEUE_SIZE = 100;
        if (this.queue.length > MAX_FALLBACK_QUEUE_SIZE) {
          // Keep only the newest 100 events
          this.queue = this.queue.slice(-MAX_FALLBACK_QUEUE_SIZE);
        }
      } catch (error) {
        console.error("[Analytics Engine] Failed to flush events", error);
        this.queue = batch.concat(this.queue);
        const MAX_FALLBACK_QUEUE_SIZE = 100;
        if (this.queue.length > MAX_FALLBACK_QUEUE_SIZE) {
          // Keep only the newest 100 events
          this.queue = this.queue.slice(-MAX_FALLBACK_QUEUE_SIZE);
        }
      }
    }
  }

  global.AnalyticsEngine = AnalyticsEngine;
})(window);
