**Summary:**
The workflow service gRPC API is now protected with authentication and authorization. Mutating endpoints require an "admin-token"; read-only endpoints allow "user-token" or "admin-token". Unit tests for access control pass. Some unrelated tests are failing due to test data, not the new security logic.

**Recent Progress:**
- Implemented and tested API security for the workflow service.
- All access control tests pass; API is now protected.

**Next Steps (candidates):**
- Implement advanced scheduling logic (priorities, load, etc.) in the workflow service.
- Develop monitoring dashboard/interface for workflow and node status.
- Enhance node service with additional agent roles and integration logic.
- Continue RFC/governance automation infrastructure.

**Open Questions / Research:** See [[research/TODO.md]].

**Blocking Issues:** Some unrelated tests are failing; review and address as needed.