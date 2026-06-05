import React, { useEffect, useState } from 'react';
import { useLocation, useHistory } from 'react-router-dom';
import { useAuth } from '../../helpers/AuthContent';

function OAuthCallback() {
    const location = useLocation();
    const history = useHistory();
    const { loginFromToken } = useAuth();
    const [error, setError] = useState('');

    useEffect(() => {
        const queryParams = new URLSearchParams(location.search);
        const token = queryParams.get('token');
        const username = queryParams.get('username');
        const usertype = queryParams.get('usertype');
        const mustChangePassword = queryParams.get('mustChangePassword') === 'true';

        if (token && username && usertype) {
            loginFromToken(token, username, usertype, mustChangePassword);
            history.replace('/markets');
        } else {
            setError('Authentication failed. Missing required credentials.');
        }
    }, [location, history, loginFromToken]);

    if (error) {
        return (
            <div className="min-h-[calc(100vh-6rem)] flex items-center justify-center">
                <div className="bg-gray-800 p-8 rounded-lg shadow-lg text-center">
                    <h2 className="text-xl text-red-500 mb-4">Error</h2>
                    <p className="text-white">{error}</p>
                    <button 
                        onClick={() => history.push('/')}
                        className="mt-6 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
                    >
                        Go Home
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-[calc(100vh-6rem)] flex items-center justify-center">
            <div className="text-white text-xl">
                Completing login...
            </div>
        </div>
    );
}

export default OAuthCallback;
