function textFrom(selector) {
  const node = document.querySelector(selector);
  return node?.textContent?.trim() || '';
}

function firstText(selectors, root = document) {
  for (const selector of selectors) {
    const node = root.querySelector(selector);
    const text = cleanText(node?.textContent || '');
    if (text) return text;
  }
  return '';
}

function cleanText(text) {
  return text
    .replace(/\r/g, '\n')
    .replace(/[ \t]+/g, ' ')
    .replace(/\n[ \t]+/g, '\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

var JOB_DETAIL_MARKERS = [
  'About the job',
  'About The Role',
  'What You Would Actually Do',
  'The Kind Of Person Who Does Well Here',
  'People Who Tend To Do Well Here',
  'How You Would Work',
  'Why Maincode',
  "The Skills You'll Need",
  'Responsibilities',
  'Requirements',
];

function markerScore(text) {
  return JOB_DETAIL_MARKERS.reduce((score, marker) => score + (text.includes(marker) ? 1 : 0), 0);
}

function isLikelyJobDetailText(text) {
  text = cleanText(text);
  if (text.includes('Top job picks for you') || text.includes('Recent job searches')) return false;
  return text.length > 300 && markerScore(text) > 0;
}

function findDetailsRoot() {
  const selectors = [
    '.jobs-search__job-details--container',
    '.jobs-search__job-details',
    '.jobs-search-two-pane__details',
    '.jobs-search__job-details--wrapper',
    '.job-view-layout',
    '.jobs-details',
    '.jobs-details__main-content',
    'main',
  ];
  const candidates = [];
  for (const selector of selectors) {
    const node = document.querySelector(selector);
    const text = cleanText(node?.innerText || node?.textContent || '');
    if (node && isLikelyJobDetailText(text)) candidates.push({node, text, rect: node.getBoundingClientRect()});
  }
  for (const y of [170, 240, 340, Math.floor(window.innerHeight * 0.5), Math.floor(window.innerHeight * 0.72)]) {
    const leaf = document.elementFromPoint(Math.floor(window.innerWidth * 0.72), y);
    let node = leaf;
    for (let depth = 0; node && depth < 10; depth += 1, node = node.parentElement) {
      const text = cleanText(node.innerText || node.textContent || '');
      if (isLikelyJobDetailText(text)) candidates.push({node, text, rect: node.getBoundingClientRect()});
    }
  }
  candidates.sort((a, b) => {
    const aRight = a.rect.left > window.innerWidth * 0.38 ? 1 : 0;
    const bRight = b.rect.left > window.innerWidth * 0.38 ? 1 : 0;
    if (aRight !== bRight) return bRight - aRight;
    const markerDiff = markerScore(b.text) - markerScore(a.text);
    if (markerDiff !== 0) return markerDiff;
    return a.text.length - b.text.length;
  });
  return candidates[0]?.node || null;
}

function visibleRightPaneText() {
  const viewportMid = window.innerWidth * 0.42;
  const candidates = Array.from(document.querySelectorAll('main, section, article, div'))
    .map((node) => {
      const rect = node.getBoundingClientRect();
      const text = cleanText(node.innerText || node.textContent || '');
      return {node, rect, text};
    })
    .filter(({rect, text}) => (
      isLikelyJobDetailText(text) &&
      text.length > 300 &&
      rect.width > 250 &&
      rect.height > 150 &&
      rect.left > viewportMid &&
      rect.bottom > 80 &&
      rect.top < window.innerHeight - 80
    ))
    .sort((a, b) => {
      const aRightBias = a.rect.left > viewportMid ? 0 : 1;
      const bRightBias = b.rect.left > viewportMid ? 0 : 1;
      if (aRightBias !== bRightBias) return aRightBias - bRightBias;
      return Math.abs(a.text.length - 5000) - Math.abs(b.text.length - 5000);
    });
  return candidates[0]?.text || '';
}

function sliceLinkedInJobText(text, requireAboutJob = false) {
  text = cleanText(text);
  const startMarkers = ['About the job', 'About The Job', 'About The Role', 'About the Role'];
  let foundStart = false;
  for (const marker of startMarkers) {
    const index = text.indexOf(marker);
    if (index >= 0) {
      text = text.slice(index + marker.length).trim();
      foundStart = true;
      break;
    }
  }
  if (requireAboutJob && !foundStart) return '';

  const endMarkers = [
    '\nJob search faster with Premium',
    '\nAbout the company',
    '\nAbout The Company',
    '\nCommitments',
    '\nInterested in working with us in the future?',
    '\nTrending employee content',
    '\nShow more',
    '\nPeople you can reach out to',
    '\nYour profile matches',
    '\nPromoted by hirer',
  ];
  let end = text.length;
  for (const marker of endMarkers) {
    const index = text.indexOf(marker);
    if (index >= 0 && index < end) end = index;
  }
  text = cleanText(text.slice(0, end));
  if (foundStart) return text.length > 200 ? text : '';
  return isLikelyJobDetailText(text) ? text : '';
}

function extractJobDescription() {
  const detailsRoot = findDetailsRoot();
  const descriptionSelectors = [
    '.jobs-description__content',
    '.jobs-box__html-content',
    '#job-details',
    '.jobs-description-content__text',
    '[data-job-id] .jobs-description-content__text',
    '.jobs-search__job-details--container',
  ];
  const titleSelectors = [
    '.job-details-jobs-unified-top-card__job-title',
    '.jobs-unified-top-card__job-title',
    'h1',
  ];
  const companySelectors = [
    '.job-details-jobs-unified-top-card__company-name',
    '.jobs-unified-top-card__company-name',
    '.job-details-jobs-unified-top-card__primary-description-container a',
  ];

  let rawText = '';
  for (const selector of descriptionSelectors) {
    const node = detailsRoot?.querySelector(selector) || document.querySelector(selector);
    rawText = sliceLinkedInJobText(node?.innerText || node?.textContent || '');
    if (rawText.length > 200) break;
  }
  if (rawText.length <= 200) {
    rawText = sliceLinkedInJobText(detailsRoot?.innerText || detailsRoot?.textContent || '');
  }
  if (rawText.length <= 200) {
    rawText = sliceLinkedInJobText(visibleRightPaneText());
  }
  if (rawText.length <= 200) {
    rawText = sliceLinkedInJobText(document.body.innerText || document.body.textContent || '', true);
  }

  return {
    title: (detailsRoot ? firstText(titleSelectors, detailsRoot) : '') || cleanText(titleSelectors.map(textFrom).find(Boolean) || document.title.replace(/\| LinkedIn.*/i, '')),
    company: (detailsRoot ? firstText(companySelectors, detailsRoot) : '') || cleanText(companySelectors.map(textFrom).find(Boolean) || ''),
    url: location.href,
    raw_text: rawText,
    warnings: rawText.length > 200 ? [] : ['Could not find the selected LinkedIn job details panel; click a job result so the details panel is visible, then import again.'],
  };
}

window.__jdTailorExtractLinkedInJob = extractJobDescription;

if (!window.__jdTailorLinkedInListenerInstalled) {
  window.__jdTailorLinkedInListenerInstalled = true;
  browser.runtime.onMessage.addListener((message) => {
    if (message?.type !== 'extract-linkedin-job') return undefined;
    const result = extractJobDescription();
    if (!result.raw_text || result.raw_text.length < 200) {
      return Promise.resolve({error: 'Click a job result so the right-side details panel is visible, then import again.'});
    }
    return Promise.resolve(result);
  });
}

function __jdTailorExtractForExecuteScript() {
  const result = extractJobDescription();
  if (!result.raw_text || result.raw_text.length < 200) {
    return {error: 'Click a job result so the right-side details panel is visible, then import again.'};
  }
  return result;
}

true;
