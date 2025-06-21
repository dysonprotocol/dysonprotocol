# Tasks for PBI 8: NFT List & Detail View with Bidding

This document lists all tasks associated with PBI 8.

**Parent PBI**: [PBI 8](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--------------------------------------------- | :------- | :-------------------------------------------------------------- |
| 8-1 | [Analyse requirements & UX flows](./8-1.md) | Done | Document detailed requirements, API design, UX wireframes, and acceptance tests |
| 8-2 | [Add backend routes in `script.py`](./8-2.md) | In Progress | Implement `/nftclasses`, `/nfts/class/{class_id}`, `/nfts/owner/{address}` routes |
| 8-3 | [Create templates `nftclasses.html`, `nfts.html`](./8-3.md) | In Progress | Build HTMX/Alpine enabled templates for class list & NFT detail views |
| 8-4 | [Implement `nftStore.js` Alpine store](./8-4.md) | Proposed | Client-side logic for fetching classes/NFTs, bidding, accept/reject, listed toggle |
| 8-5 | [Integrate navigation & explanatory notes](./8-5.md) | Proposed | Add navigation link, ensure note about `nftclass.always_listed` priority |
| 8-6 | [End-to-end tests for bidding & listing](./8-6.md) | Proposed | Write E2E tests validating AC1–AC7 |

## Task History Log

| Timestamp (YYYYMMDD-HHMMSS) | Task ID | Change Description |
|---|---|---| 
| 20250621-121000 | 8-1 | Status to Done | Analysis completed, skeleton doc created | User |
| 20250621-121000 | 8-2 | Status to In Progress | Backend routes implementation started | User |
| 20250621-121200 | 8-3 | Status to In Progress | Started scaffolding templates | User | 