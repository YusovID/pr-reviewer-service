import http from 'k6/http';
import { check, group, sleep } from 'k6';

const BASE_URL = 'http://localhost:8080';

export const options = {
    thresholds: {
        'http_req_duration': ['p(95)<300'],
        'http_req_failed': ['rate<0.01'],
    },

    scenarios: {
        main_flow: {
            executor: 'constant-arrival-rate',
            rate: 5,
            timeUnit: '1s',
            duration: '30s',
            preAllocatedVUs: 10,
            maxVUs: 50,
        },
    },
};

export default function () {
    const uniqueId = `${__VU}-${__ITER}`;
    const teamName = `loadtest-team-${uniqueId}`;
    const prId = `pr-${uniqueId}`;
    const authorId = `u-author-${uniqueId}`;
    const reviewerIds = [`u-rev1-${uniqueId}`, `u-rev2-${uniqueId}`];

    const headers = { 'Content-Type': 'application/json' };

    group('1. Create Team', function () {
        const payload = JSON.stringify({
            team_name: teamName,
            members: [
                { user_id: authorId, username: `Author ${uniqueId}`, is_active: true },
                { user_id: reviewerIds[0], username: `Reviewer 1 ${uniqueId}`, is_active: true },
                { user_id: reviewerIds[1], username: `Reviewer 2 ${uniqueId}`, is_active: true },
            ],
        });

        const res = http.post(`${BASE_URL}/team/add`, payload, { headers });
        check(res, { 'Team created successfully (status 201)': (r) => r.status === 201 });
    });

    group('2. Create Pull Request', function () {
        const payload = JSON.stringify({
            pull_request_id: prId,
            pull_request_name: `Feature PR ${uniqueId}`,
            author_id: authorId,
        });

        const res = http.post(`${BASE_URL}/pullRequest/create`, payload, { headers });
        check(res, { 'PR created successfully (status 201)': (r) => r.status === 201 });
    });

    group('3. Get Review Assignments', function () {
        const res = http.get(`${BASE_URL}/users/getReview?user_id=${reviewerIds[0]}`);
        check(res, { 'Reviews fetched successfully (status 200)': (r) => r.status === 200 });
    });

    sleep(1);
}