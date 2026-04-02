<script setup>
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="min-h-screen py-16 px-6">
    <div class="max-w-[800px] mx-auto">
      <h1 class="font-[Cinzel] text-[36px] font-bold text-[#e8e8f0] mb-2">Privacy Policy</h1>
      <p class="text-[#7ec8e3]/50 text-sm mb-10">Last updated: April 2, 2026</p>

      <div class="space-y-8 text-[#e8e8f0]/80 text-[15px] leading-relaxed">
        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">1. Information We Collect</h2>

          <h3 class="text-[16px] font-semibold text-[#e8e8f0] mt-4 mb-2">Account Information</h3>
          <p>When you create an account, we collect your email address, username, and password (stored in hashed form). You may optionally provide a display name and profile avatar.</p>

          <h3 class="text-[16px] font-semibold text-[#e8e8f0] mt-4 mb-2">User-Uploaded Content</h3>
          <p>You may upload custom artwork and images (such as character portraits, game covers, and map assets) to use within your games. These images are stored on our servers and are only accessible to participants in your games. We do not use, sell, or share your uploaded artwork with any third parties.</p>

          <h3 class="text-[16px] font-semibold text-[#e8e8f0] mt-4 mb-2">Payment Information</h3>
          <p>If you subscribe to a paid plan, your payment is processed by a third-party payment provider. We do not store your full credit card number, CVV, or other sensitive payment details on our servers. We may store a payment reference ID, subscription status, and billing dates for account management purposes.</p>

          <h3 class="text-[16px] font-semibold text-[#e8e8f0] mt-4 mb-2">Usage Data</h3>
          <p>We may collect basic usage data such as login timestamps, pages visited, and feature usage patterns to improve the Service.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">2. How We Use Your Information</h2>
          <ul class="list-disc list-inside space-y-2 ml-2">
            <li>To provide, maintain, and improve the Service</li>
            <li>To authenticate your identity and manage your account</li>
            <li>To process subscription payments</li>
            <li>To send important account notifications (verification, security alerts)</li>
            <li>To respond to support requests</li>
          </ul>
          <p class="mt-3">We do not sell your personal information to third parties. We do not use your data for advertising or marketing purposes beyond service-related communications.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">3. Data Storage and Security</h2>
          <p>Your data is stored on secure servers. Passwords are cryptographically hashed and never stored in plain text. We use HTTPS encryption for all data transmitted between your browser and our servers. We implement access controls to ensure only authorized personnel can access user data.</p>
          <p class="mt-3">Uploaded images are stored in our file system and served only to authenticated users who are participants in the relevant games.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">4. Data Sharing</h2>
          <p>We share your data only in the following limited circumstances:</p>
          <ul class="list-disc list-inside space-y-2 ml-2 mt-2">
            <li><strong>Payment processing:</strong> Subscription payment data is shared with our payment processor to complete transactions</li>
            <li><strong>Legal requirements:</strong> We may disclose data if required by law or legal process</li>
            <li><strong>Game participants:</strong> Your username and avatar are visible to other members of games you join</li>
          </ul>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">5. Your Rights</h2>
          <p>You have the right to:</p>
          <ul class="list-disc list-inside space-y-2 ml-2 mt-2">
            <li>Access your personal data through your account settings</li>
            <li>Update or correct your personal information at any time</li>
            <li>Delete your account and all associated data</li>
            <li>Export your data upon request</li>
            <li>Withdraw consent for data processing (this may limit your ability to use the Service)</li>
          </ul>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">6. Data Retention</h2>
          <p>We retain your account data for as long as your account is active. If you delete your account, we will remove your personal data and uploaded content from our servers within 30 days. Some data may be retained longer if required by law or for legitimate business purposes (e.g., billing records).</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">7. Cookies</h2>
          <p>We use essential cookies and local storage to maintain your authentication session. We do not use tracking cookies, analytics cookies, or third-party advertising cookies.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">8. Children's Privacy</h2>
          <p>The Service is not intended for children under 13 years of age. We do not knowingly collect personal information from children under 13. If we learn that we have collected data from a child under 13, we will promptly delete it.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">9. Changes to This Policy</h2>
          <p>We may update this Privacy Policy from time to time. We will notify registered users of material changes via email. The "Last updated" date at the top indicates when the policy was last revised.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">10. Contact</h2>
          <p>If you have questions about this Privacy Policy or wish to exercise your data rights, contact us at <a href="mailto:privacy@dogmaliter.com" class="text-[#e94560] hover:underline">privacy@dogmaliter.com</a>.</p>
        </section>
      </div>
    </div>
  </div>
</template>
