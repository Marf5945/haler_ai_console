(() => {
  const selectors = [
    "button",
    "a[href]",
    "[role='button']",
    "input[type='button']",
    "input[type='submit']",
    "input[type='reset']"
  ];

  const round = (value) => Math.round(value * 100) / 100;

  const textOf = (el) => {
    if (el instanceof HTMLInputElement) return el.value || el.getAttribute("aria-label") || "";
    return (el.innerText || el.textContent || el.getAttribute("aria-label") || "").trim();
  };

  const hasButtonChrome = (style, rect) => {
    const bg = style.backgroundColor && style.backgroundColor !== "rgba(0, 0, 0, 0)";
    const border = parseFloat(style.borderTopWidth || "0") > 0;
    const radius = parseFloat(style.borderTopLeftRadius || "0") > 0;
    const padded = parseFloat(style.paddingLeft || "0") + parseFloat(style.paddingRight || "0") >= 8;
    return bg || border || radius || (padded && rect.height <= 56);
  };

  const cssLooksButtonLike = (style, rect, text) => {
    const shortText = text.length > 0 && text.length <= 40;
    return hasButtonChrome(style, rect) && shortText && rect.height >= 16;
  };

  const isCovered = (el, rect) => {
    const points = [
      [rect.left + rect.width / 2, rect.top + rect.height / 2],
      [rect.left + Math.min(rect.width - 1, 6), rect.top + Math.min(rect.height - 1, 6)],
      [rect.right - Math.min(rect.width - 1, 6), rect.bottom - Math.min(rect.height - 1, 6)]
    ];

    return points.every(([x, y]) => {
      if (x < 0 || y < 0 || x >= window.innerWidth || y >= window.innerHeight) return true;
      const top = document.elementFromPoint(x, y);
      return top && top !== el && !el.contains(top) && !top.contains(el);
    });
  };

  const seen = new Set();
  const candidates = Array.from(document.querySelectorAll(selectors.join(",")))
    .map((el, index) => {
      const rect = el.getBoundingClientRect();
      const style = window.getComputedStyle(el);
      const text = textOf(el);
      const visible =
        rect.width > 4 &&
        rect.height > 4 &&
        style.display !== "none" &&
        style.visibility !== "hidden" &&
        Number(style.opacity) > 0.05 &&
        rect.bottom > 0 &&
        rect.right > 0 &&
        rect.top < window.innerHeight &&
        rect.left < window.innerWidth;

      const clipped = {
        left: Math.max(0, rect.left),
        top: Math.max(0, rect.top),
        right: Math.min(window.innerWidth, rect.right),
        bottom: Math.min(window.innerHeight, rect.bottom)
      };
      clipped.width = Math.max(0, clipped.right - clipped.left);
      clipped.height = Math.max(0, clipped.bottom - clipped.top);

      const areaRatio = (clipped.width * clipped.height) / (window.innerWidth * window.innerHeight);
      const disabled = Boolean(
        el.disabled ||
          el.getAttribute("disabled") !== null ||
          el.getAttribute("aria-disabled") === "true"
      );
      const covered = visible ? isCovered(el, rect) : false;
      const tag = el.tagName.toLowerCase();
      const role = el.getAttribute("role") || "";
      const isAnchor = tag === "a";
      const isMediaAnchor =
        isAnchor &&
        el instanceof HTMLAnchorElement &&
        (el.pathname.startsWith("/watch") || el.pathname.startsWith("/shorts/")) &&
        (clipped.width > 120 || clipped.height > 80);
      const isLargeAnchor = isAnchor && (areaRatio > 0.035 || clipped.width > 220 || clipped.height > 88);
      const isLongTextAnchor = isAnchor && text.length > 24 && !hasButtonChrome(style, clipped);
      const anchorLooksButton =
        isAnchor &&
        !isMediaAnchor &&
        !isLargeAnchor &&
        !isLongTextAnchor &&
        cssLooksButtonLike(style, clipped, text);
      const looksButtonLike =
        tag === "button" ||
        role === "button" ||
        role === "tab" ||
        tag === "input" ||
        anchorLooksButton;

      const confidenceRule =
        visible && !covered && areaRatio <= 0.08 && looksButtonLike ? "high" : "low";

      return {
        id: `dom_${String(index + 1).padStart(3, "0")}`,
        tag,
        role,
        text,
        href: el instanceof HTMLAnchorElement ? el.href : "",
        rect: [round(clipped.left), round(clipped.top), round(clipped.width), round(clipped.height)],
        visible,
        disabled,
        covered,
        area_ratio: round(areaRatio),
        skip_reason: isMediaAnchor
          ? "media_anchor"
          : isLargeAnchor
            ? "large_anchor"
            : isLongTextAnchor
              ? "long_text_anchor"
              : "",
        class_guess: "button",
        confidence_rule: confidenceRule,
        source: "dom_auto",
        review_status: "pending"
      };
    })
    .filter((item) => {
      const key = item.rect.join(",");
      if (seen.has(key)) return false;
      seen.add(key);
      return item.rect[2] >= 8 && item.rect[3] >= 8 && item.visible && item.confidence_rule === "high";
    });

  const result = {
    page: {
      url: location.href,
      title: document.title,
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight,
        device_pixel_ratio: window.devicePixelRatio
      },
      captured_at: new Date().toISOString()
    },
    candidates
  };

  const json = JSON.stringify(result, null, 2);
  window.__YOLO_DOM_JSON__ = json;
  if (typeof copy === "function") copy(json);
  console.log(`YOLO DOM JSON ready: ${candidates.length} candidates. If paste fails, run copy(__YOLO_DOM_JSON__)`);
  console.log(result);
  return result;
})();
