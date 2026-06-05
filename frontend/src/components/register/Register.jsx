import React, { useState } from 'react';
import { useHistory } from 'react-router-dom';
import { PersonInput, LockInput, EmailInput } from '../inputs/InputBar';
import SiteButton from '../buttons/SiteButtons';
import { useAuth } from '../../helpers/AuthContent';
import useFrontendConfig from '../../hooks/useFrontendConfig';
import { apiRequest } from '../../api/httpClient';

function Register() {
    const [username, setUsername] = useState('');
    const [displayname, setDisplayname] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [error, setError] = useState('');
    const [isSubmitting, setIsSubmitting] = useState(false);
    
    const history = useHistory();
    const { login, isLoggedIn } = useAuth();
    const { frontendConfig } = useFrontendConfig();

    if (isLoggedIn) {
        history.push('/markets');
        return null;
    }

    const oauthGoogleEnabled = frontendConfig?.oauthProviders?.google;

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');

        if (password !== confirmPassword) {
            setError("Passwords do not match");
            return;
        }

        setIsSubmitting(true);
        try {
            const response = await apiRequest('/v0/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, displayname, email, password }),
                unwrap: false // We need the raw envelope response to check response.ok
            });

            if (response.ok) {
                // Auto-login after registration
                const loginResult = await login(username, password);
                if (loginResult?.success) {
                    history.push('/markets');
                }
            } else {
                setError(response.reason || "Registration failed");
            }
        } catch (err) {
            setError(err.message || "An error occurred during registration");
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleGoogleLogin = () => {
        window.location.href = '/v0/auth/login/google';
    };

    return (
        <div className="min-h-[calc(100vh-6rem)] bg-primary-background text-custom-gray-verylight flex flex-col justify-center py-8 px-4 sm:px-6 lg:px-8">
            <div className="max-w-md mx-auto w-full">
                <div className="bg-gray-800 p-8 rounded-lg shadow-lg">
                    <h2 className="text-2xl font-bold text-center mb-6 text-white">Create an Account</h2>
                    
                    <form onSubmit={handleSubmit} className="space-y-4">
                        <PersonInput 
                            value={username}
                            placeholder="Username (must be lowercase)"
                            onChange={(e) => setUsername(e.target.value)} 
                        />
                        <div className="flex items-center border-2 border-blue-500 bg-transparent rounded-md">
                            <span className="h-5 w-5 text-blue-500 ml-2">👤</span>
                            <input
                                type="text"
                                placeholder="Display Name"
                                value={displayname}
                                onChange={(e) => setDisplayname(e.target.value)}
                                className="flex-1 px-4 py-2 rounded-md text-white bg-transparent focus:outline-none"
                            />
                        </div>
                        <EmailInput 
                            value={email} 
                            onChange={(e) => setEmail(e.target.value)} 
                        />
                        <LockInput 
                            value={password} 
                            onChange={(e) => setPassword(e.target.value)} 
                        />
                        <p id="new-password-requirements" className="text-sm text-gray-400 px-2">
                            Use 8-128 characters with uppercase, lowercase, and a digit.
                        </p>
                        <div className="flex items-center border-2 border-blue-500 bg-transparent rounded-md">
                            <span className="h-5 w-5 text-blue-500 ml-2">🔒</span>
                            <input
                                type="password"
                                placeholder="Confirm Password"
                                value={confirmPassword}
                                onChange={(e) => setConfirmPassword(e.target.value)}
                                className="flex-1 px-4 py-2 rounded-md text-white bg-transparent focus:outline-none"
                            />
                        </div>
                        
                        {error && <div className="text-red-500 text-sm text-center">{error}</div>}
                        
                        <div className="pt-2">
                            <SiteButton type="submit" disabled={isSubmitting} className="w-full justify-center">
                                {isSubmitting ? 'Registering...' : 'Register'}
                            </SiteButton>
                        </div>
                    </form>

                    {oauthGoogleEnabled && (
                        <div className="mt-6">
                            <div className="relative">
                                <div className="absolute inset-0 flex items-center">
                                    <div className="w-full border-t border-gray-600"></div>
                                </div>
                                <div className="relative flex justify-center text-sm">
                                    <span className="px-2 bg-gray-800 text-gray-400">Or continue with</span>
                                </div>
                            </div>
                            
                            <div className="mt-6">
                                <button
                                    onClick={handleGoogleLogin}
                                    className="w-full flex justify-center items-center px-4 py-2 border border-gray-600 rounded-md shadow-sm text-sm font-medium text-white bg-gray-700 hover:bg-gray-600 transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-800 focus:ring-blue-500"
                                >
                                    <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24">
                                        <path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                                        <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                                        <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
                                        <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
                                        <path fill="none" d="M1 1h22v22H1z" />
                                    </svg>
                                    Sign in with Google
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

export default Register;