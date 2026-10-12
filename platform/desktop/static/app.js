const autoButton = document.getElementById("auto");
const manualButton = document.getElementById("manual");
const proxyField = document.getElementById("proxyWrap");
const proxyInput = document.getElementById("proxy");
const excludeInput = document.getElementById("exclude");
const exeInput = document.getElementById("exe");
const connectButton = document.getElementById("connect");
const quitButton = document.getElementById("quit");
const koreanButton = document.getElementById("langKo");
const englishButton = document.getElementById("langEn");
const status = document.getElementById("status");
const headline = document.getElementById("headline");
const result = document.getElementById("result");

let language = localStorage.getItem("bycord-language");
if(language !== "ko" && language !== "en") {
    language = navigator.language.toLowerCase().startsWith("ko") ? "ko" : "en";
}

function text(key) {
    return translations[language][key];
}

function setLanguage(selected) {
    language = selected;
    localStorage.setItem("bycord-language", language);
    document.documentElement.lang = language;
    document.querySelectorAll("[data-i18n]").forEach(element => {
        element.textContent = text(element.dataset.i18n);
    });
    document.querySelectorAll("[data-i18n-placeholder]").forEach(element => {
        element.placeholder = text(element.dataset.i18nPlaceholder);
    });
    koreanButton.classList.toggle("on", language === "ko");
    englishButton.classList.toggle("on", language === "en");
    koreanButton.setAttribute("aria-pressed", language === "ko");
    englishButton.setAttribute("aria-pressed", language === "en");
}

function setMode(automatic) {
    autoButton.classList.toggle("on", automatic);
    manualButton.classList.toggle("on", !automatic);
    autoButton.setAttribute("aria-selected", automatic);
    manualButton.setAttribute("aria-selected", !automatic);
    proxyField.hidden = automatic;
}

function showStatus(kind, title, detail) {
    status.className = `status show ${kind}`.trim();
    headline.textContent = title;
    result.textContent = detail;
}

function localizeError(message) {
    const value = message.toLowerCase();
    if(value.includes("excluded country") || message.includes("제외 국가")) return text("invalidCountry");
    if(value.includes("proxy address") || message.includes("프록시 주소")) return text("invalidProxy");
    if(value.includes("no proxy") || value.includes("proxy candidates")) return text("noProxy");
    if(value.includes("discord was not found") || value.includes("not found in path")) return text("discordNotFound");
    return message;
}

async function loadDefaultCountry() {
    let country = "KR";
    try {
        const response = await fetch("country");
        const data = await response.json();
        if(response.ok && /^[A-Z]{2}$/.test(data.country)) country = data.country;
    } catch(error) {
    }
    if(excludeInput.value.trim() === "") excludeInput.value = country;
}

connectButton.addEventListener("click", async () => {
    await loadDefaultCountry();
    connectButton.disabled = true;
    status.className = "status show";
    headline.innerHTML = `<span class="pulse"></span>${text("connecting")}`;
    result.textContent = text("connectingDetail");

    const request = {
        proxy: proxyField.hidden ? "" : proxyInput.value.trim(),
        exclude: excludeInput.value.trim(),
        exe: exeInput.value.trim(),
        language,
    };

    try {
        const response = await fetch("connect", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(request),
        });
        const data = await response.json();
        if(!response.ok) {
            throw new Error(data.error || text("failedDetail"));
        }
        showStatus(
            "ok",
            text("connected"),
            `${data.endpoint} · ${data.country} · ${data.latency} ms\n${text("started")}`,
        );
    } catch(error) {
        showStatus("bad", text("failed"), localizeError(error.message));
    } finally {
        connectButton.disabled = false;
    }
});

quitButton.addEventListener("click", async () => {
    await fetch("quit", { method: "POST" });
    document.body.innerHTML = `
    <main class="shell">
        <section class="card">
            <h2>${text("stopped")}</h2>
            <p class="notice">${text("closeWindow")}</p>
        </section>
    </main>`;
});

koreanButton.addEventListener("click", () => setLanguage("ko"));
englishButton.addEventListener("click", () => setLanguage("en"));
autoButton.addEventListener("click", () => setMode(true));
manualButton.addEventListener("click", () => setMode(false));
setLanguage(language);