// Mock data standing in for a real backend. Replace with API calls
// (e.g. GET /api/investigations, GET /api/investigations/:id) later.

export const INVESTIGATIONS = [
  {
    id: 'inv-2025-0421',
    title: 'Contract Signature Verification',
    fileName: 'Contract_Signed.pdf',
    caseId: 'INV-2025-0421',
    type: 'Signature Verification',
    submittedDate: 'May 23, 2025',
    submittedTime: '10:24 AM',
    status: 'Submitted',
    priority: 'High',
    lastUpdated: 'May 23, 2025 \u2022 10:24 AM',
    lastUpdatedBy: 'Marie Uwimana',
  },
  {
    id: 'inv-2025-0419',
    title: 'Invoice Authenticity Check',
    fileName: 'Invoice_Jan2025.pdf',
    caseId: 'INV-2025-0419',
    type: 'Document Authenticity',
    submittedDate: 'May 22, 2025',
    submittedTime: '09:40 AM',
    status: 'Under Review',
    priority: 'Normal',
    lastUpdated: 'May 23, 2025 \u2022 11:15 AM',
    lastUpdatedBy: 'Yves Seraphin',
  },
  {
    id: 'inv-2025-0418',
    title: 'Purchase Order Tampering',
    fileName: 'PO_450023.pdf',
    caseId: 'INV-2025-0418',
    type: 'Suspected Tampering',
    submittedDate: 'May 21, 2025',
    submittedTime: '02:11 PM',
    status: 'Action Required',
    priority: 'Urgent',
    lastUpdated: 'May 23, 2025 \u2022 09:02 AM',
    lastUpdatedBy: 'Kevin Niyonsaba',
  },
  {
    id: 'inv-2025-0417',
    title: 'NDA Signature Verification',
    fileName: 'NDA_Signed.pdf',
    caseId: 'INV-2025-0417',
    type: 'Signature Verification',
    submittedDate: 'May 20, 2025',
    submittedTime: '11:05 AM',
    status: 'Completed',
    priority: 'Normal',
    lastUpdated: 'May 22, 2025 \u2022 04:30 PM',
    lastUpdatedBy: 'Marie Uwimana',
  },
  {
    id: 'inv-2025-0415',
    title: 'Bank Statement Review',
    fileName: 'Bank_Statement_Apr.pdf',
    caseId: 'INV-2025-0415',
    type: 'Other Forensic Review',
    submittedDate: 'May 19, 2025',
    submittedTime: '03:22 PM',
    status: 'Closed',
    priority: 'Normal',
    lastUpdated: 'May 21, 2025 \u2022 01:10 PM',
    lastUpdatedBy: 'Yves Seraphin',
  },
  {
    id: 'inv-2025-0412',
    title: 'Employment Agreement Check',
    fileName: 'Employment_Agreement.pdf',
    caseId: 'INV-2025-0412',
    type: 'Signature Verification',
    submittedDate: 'May 15, 2025',
    submittedTime: '10:18 AM',
    status: 'Draft',
    priority: 'Normal',
    lastUpdated: 'May 15, 2025 \u2022 10:18 AM',
    lastUpdatedBy: 'Yves Seraphin',
  },
  {
    id: 'case-2026-00124',
    title: 'Signature Authenticity Investigation',
    fileName: 'Contract_Signed.pdf',
    caseId: 'CASE-2026-00124',
    type: 'Signature Verification',
    submittedDate: 'Aug 20, 2026',
    submittedTime: '10:42 AM',
    status: 'Under Review',
    priority: 'High',
    lastUpdated: 'Aug 22, 2026 \u2022 02:15 PM',
    lastUpdatedBy: 'Mark Ferdinand',
  },
]

export function getInvestigationById(id) {
  return INVESTIGATIONS.find((inv) => inv.id === id)
}

// Rich detail data only exists for cases that have actually been
// processed. Anything not listed here falls back to a simpler view
// (see AnalysisTab / DocumentsTab in InvestigationDetail.jsx).
export const CASE_DETAILS = {
  'case-2026-00124': {
    description:
      'The client suspects that the signature on the attached contract may be forged. Please verify the authenticity of the signature by comparing it with the provided reference signatures.',
    timeline: [
      { label: 'Investigation submitted', date: 'Aug 20, 2026 \u2014 10:42 AM', done: true },
      { label: 'Documents received', date: 'Aug 20, 2026 \u2014 10:43 AM', done: true },
      { label: 'Assigned to forensic examiner', date: 'Aug 21, 2026 \u2014 09:30 AM', done: true },
      { label: 'Examination in progress', date: 'Aug 22, 2026 \u2014 11:05 AM', done: false },
    ],
    documents: {
      questioned: [
        { name: 'Contract_Signed.pdf', kind: 'PDF', size: '2.45 MB', date: 'Aug 20, 2026 \u2022 10:42 AM' },
      ],
      reference: [
        { name: 'Reference_Signature_1.png', kind: 'PNG', size: '320 KB', date: 'Aug 19, 2026 \u2022 04:15 PM' },
        { name: 'Reference_Signature_2.jpg', kind: 'JPG', size: '280 KB', date: 'Aug 19, 2026 \u2022 04:16 PM' },
      ],
      supporting: [
        { name: 'ID_Card_Scan.jpg', kind: 'JPG', size: '1.12 MB', date: 'Aug 20, 2026 \u2022 10:45 AM' },
      ],
    },
    analysis: {
      similarityScore: 82,
      similarityLabel: 'High Similarity',
      forgeryRisk: 'Low',
      referenceMatch: 91,
      referenceMatchFile: 'Reference_Signature_1.png',
      confidence: 'High',
      confidenceNote: 'The analysis has high confidence based on data quality and consistency.',
      features: [
        { feature: 'Stroke Pattern', score: 0.86, assessment: 'High Match' },
        { feature: 'Stroke Pressure', score: 0.79, assessment: 'High Match' },
        { feature: 'Stroke Direction', score: 0.85, assessment: 'High Match' },
        { feature: 'Shape Consistency', score: 0.81, assessment: 'High Match' },
        { feature: 'Proportion & Size', score: 0.76, assessment: 'Good Match' },
      ],
      metrics: [
        { label: 'Cosine Similarity', value: '0.82', note: 'High' },
        { label: 'Euclidean Distance', value: '12.45', note: 'Low' },
        { label: 'Dynamic Time Warping', value: '0.18', note: 'Low' },
        { label: 'Structural Similarity', value: '0.84', note: 'High' },
      ],
      observation:
        'The questioned signature exhibits strong consistency in stroke patterns, pressure distribution, and overall shape compared to the reference signatures. No significant anomalies detected.',
    },
  },
}