import http from 'k6/http';
import { check, group, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://213.165.48.130:8083';

export const options = {
    thresholds: {
        'http_req_duration': ['p(95)<2000'],
        'http_req_failed': ['rate<0.01'],
    },
    scenarios: {
        main_flow: {
            executor: 'constant-arrival-rate',
            rate: 5,
            timeUnit: '1s',
            duration: '1m',
            preAllocatedVUs: 10,
            maxVUs: 50,
        },
    },
};

function randomString(length) {
    let result = '';
    const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    const charactersLength = characters.length;
    for (let i = 0; i < length; i++) {
        result += characters.charAt(Math.floor(Math.random() * charactersLength));
    }
    return result;
}

export default function () {
    const shortId = randomString(6);

    const teamName = `tm-${shortId}`;
    const prId = `pr-${shortId}`;
    const authorId = `u1-${shortId}`;
    const reviewerId = `u2-${shortId}`;

    const headers = { 'Content-Type': 'application/json' };

    group('1. Create Team', function () {
        const payload = JSON.stringify({
            team_name: teamName,
            members: [
                { user_id: authorId, username: `Auth-${shortId}`, is_active: true },
                { user_id: reviewerId, username: `Rev-${shortId}`, is_active: true },
            ],
        });

        const res = http.post(`${BASE_URL}/team/add`, payload, { headers });

        if (res.status !== 201) {
            console.error(`❌ Team Error (${res.status}): ${res.body}`);
        }

        check(res, { 'Team created (201)': (r) => r.status === 201 });
    });

    group('2. Create Pull Request', function () {
        const payload = JSON.stringify({
            pull_request_id: prId,
            pull_request_name: `Feat-${shortId}`,
            author_id: authorId,
        });

        const res = http.post(`${BASE_URL}/pullRequest/create`, payload, { headers });

        if (res.status !== 201) {
            if (res.status !== 404) console.error(`❌ PR Error (${res.status}): ${res.body}`);
        }

        check(res, { 'PR created (201)': (r) => r.status === 201 });
    });

    group('3. Get Review Assignments', function () {
        const res = http.get(`${BASE_URL}/users/getReview?user_id=${reviewerId}`);
        check(res, { 'Reviews fetched (200)': (r) => r.status === 200 });
    });

    sleep(1);
}