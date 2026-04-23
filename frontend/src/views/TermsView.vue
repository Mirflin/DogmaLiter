<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
const router = useRouter()
const auth = useAuthStore()
function goBack() {
  router.push(auth.isAuthenticated ? '/dashboard' : '/')
}
</script>

<template>
  <div class="min-h-screen py-16 px-6">
    <div class="max-w-[800px] mx-auto">
      <button
        @click="goBack"
        class="flex items-center gap-2 text-[#7ec8e3]/50 hover:text-[#e8e8f0] bg-transparent border-none cursor-pointer text-[14px] mb-8 transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
        </svg>
        Back
      </button>
      <h1 class="font-[Cinzel] text-[36px] font-bold text-[#e8e8f0] mb-2">Terms of Service</h1>
      <p class="text-[#7ec8e3]/50 text-sm mb-10">Last updated: April 2, 2026</p>

      <div class="space-y-8 text-[#e8e8f0]/80 text-[15px] leading-relaxed">
        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">1. Acceptance of Terms</h2>
          <p>By accessing or using DogmaLiter ("the Service"), you agree to be bound by these Terms of Service. If you do not agree to these terms, you may not use the Service.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">2. Description of Service</h2>
          <p>DogmaLiter is an online tabletop role-playing game platform that allows users to create, manage, and participate in tabletop RPG sessions. The Service includes game management tools, character creation, chat functionality, and related features.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">3. User Accounts</h2>
          <p>You must register an account to use the Service. You are responsible for maintaining the confidentiality of your account credentials. You must provide accurate information during registration and keep it up to date. You must be at least 13 years of age to create an account.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">4. User Content</h2>
          <p>You retain ownership of all content you upload to the Service, including custom artwork, character images, and game assets. By uploading content, you grant DogmaLiter a limited, non-exclusive license to store, display, and serve that content solely within the platform for the purpose of providing the Service to you and other game participants.</p>
          <p class="mt-3">Your uploaded artwork and images will not be publicly displayed outside of the games you participate in, shared with third parties, or used for any purpose other than providing the Service.</p>
          <p class="mt-3">You represent that you have the right to upload any content you submit and that it does not infringe on any third party's intellectual property rights.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">5. Prohibited Conduct</h2>
          <ul class="list-disc list-inside space-y-2 ml-2">
            <li>Uploading illegal, harmful, or offensive content</li>
            <li>Attempting to gain unauthorized access to other accounts or systems</li>
            <li>Interfering with the operation of the Service</li>
            <li>Using the Service for any unlawful purpose</li>
            <li>Uploading content that infringes on copyrights, trademarks, or other rights</li>
            <li>Creating multiple accounts to circumvent restrictions</li>
          </ul>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">6. Subscriptions and Payments</h2>
          <p>Some features of the Service require a paid subscription. Subscription fees are billed in advance on a recurring basis. You may cancel your subscription at any time; cancellation takes effect at the end of the current billing period. Refunds are handled in accordance with applicable law.</p>
          <p class="mt-3">We reserve the right to change subscription pricing with 30 days' prior notice. Continued use of the Service after a price change constitutes acceptance of the new pricing.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">7. Termination</h2>
          <p>We may suspend or terminate your account if you violate these Terms. You may delete your account at any time. Upon termination, your uploaded content will be deleted from our servers within 30 days.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">8. Disclaimer of Warranties</h2>
          <p>The Service is provided "as is" without warranties of any kind, express or implied. We do not guarantee uninterrupted or error-free operation of the Service.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">9. Limitation of Liability</h2>
          <p>To the maximum extent permitted by law, DogmaLiter shall not be liable for any indirect, incidental, or consequential damages arising from your use of the Service.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">10. Changes to Terms</h2>
          <p>We may update these Terms at any time. We will notify registered users of material changes via email. Continued use of the Service after changes take effect constitutes acceptance of the updated Terms.</p>
        </section>

        <section>
          <h2 class="font-[Cinzel] text-[20px] font-bold text-[#e94560] mb-3">11. Contact</h2>
          <p>For questions about these Terms, please contact us at <a href="mailto:support@dogmaliter.com" class="text-[#e94560] hover:underline">support@dogmaliter.com</a>.</p>
        </section>
      </div>
    </div>
  </div>
</template>
