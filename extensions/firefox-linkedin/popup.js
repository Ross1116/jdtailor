const button = document.getElementById('import');
const statusEl = document.getElementById('status');

function setStatus(message) {
  statusEl.textContent = message;
}

async function extractFromTab(tabID) {
  await browser.tabs.executeScript(tabID, {file: 'content.js'});
  const results = await browser.tabs.executeScript(tabID, {code: 'window.__jdTailorExtractLinkedInJob ? window.__jdTailorExtractLinkedInJob() : {error: "Extractor did not load."}'});
  return results?.[0];
}

button.addEventListener('click', async () => {
  button.disabled = true;
  setStatus('Extracting job...');
  try {
    const tabs = await browser.tabs.query({active: true, currentWindow: true});
    const tab = tabs[0];
    if (!tab?.id || !tab.url?.includes('linkedin.com/jobs')) {
      throw new Error('Open a LinkedIn job page first.');
    }
    const response = await extractFromTab(tab.id);
    if (!response?.raw_text) throw new Error(response?.error || 'No job description found.');
    setStatus('Sending to JD Tailor...');
    const result = await fetch('http://127.0.0.1:38616/job', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({...response, source: 'firefox_linkedin_extension'}),
    });
    if (!result.ok) throw new Error(await result.text());
    setStatus('Imported. Open JD Tailor to review and parse.');
  } catch (error) {
    setStatus(error instanceof Error ? error.message : String(error));
  } finally {
    button.disabled = false;
  }
});
